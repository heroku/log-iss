package googlegoauth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/kr/session"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/technoweenie/grohl"
	"github.com/heroku/log-iss/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/heroku/log-iss/Godeps/_workspace/src/golang.org/x/oauth2"
	"github.com/heroku/log-iss/Godeps/_workspace/src/golang.org/x/oauth2/google"
)

const callbackPath = "/oauth2callback"

var Endpoint = google.Endpoint

type Session struct {
	// Client is an HTTP client obtained from oauth2.Config.Client.
	// It adds necessary OAuth2 credentials to outgoing requests to
	// perform Heroku API calls.
	*http.Client
}

type contextKey int

const sessionKey contextKey = 0

// GetSession returns data about the logged-in user
// given the Context provided to a ContextHandler.
func GetSession(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionKey).(*Session)
	return s, ok
}

// A ContextHandler can be used as the HTTP handler
// in a Handler value in order to obtain information
// about the logged-in Heroku user through the provided
// Context. See GetSession.
type ContextHandler interface {
	ServeHTTPContext(context.Context, http.ResponseWriter, *http.Request)
}

// Handler is an HTTP handler that requires
// users to log in with Heroku OAuth and requires
// them to be members of the given org.
type Handler struct {
	// RequireDomain is a domain
	// users will be required to be in.
	// If unset, any user will be permitted.
	RequireDomain string

	// Used to initialize corresponding fields of a session Config.
	// See github.com/kr/session.
	// Key should be a 64-character hex string
	// If Name is empty, "herokugoauth" is used.
	Name   string
	Path   string
	Domain string
	MaxAge time.Duration
	Key    string

	// Used to initialize corresponding fields of oauth2.Config.
	// Scopes can be nil, in which case user:email and read:org
	// will be requested.
	ClientID     string
	ClientSecret string
	Scopes       []string

	// Handler is the HTTP handler called
	// once authentication is complete.
	// If nil, http.DefaultServeMux is used.
	// If the value implements ContextHandler,
	// its ServeHTTPContext method will be called
	// instead of ServeHTTP, and a *Session value
	// can be obtained from GetSession.
	Handler http.Handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.ServeHTTPContext(context.Background(), w, r)
}

func (h *Handler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler := h.Handler
	if handler == nil {
		handler = http.DefaultServeMux
	}
	if ctx, ok := h.loginOk(ctx, w, r); ok {
		if h2, ok := handler.(ContextHandler); ok {
			h2.ServeHTTPContext(ctx, w, r)
		} else {
			handler.ServeHTTP(w, r)
		}
	}
}

// loginOk checks that the user is logged in and authorized.
// If not, it performs one step of the oauth process.
func (h *Handler) loginOk(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, bool) {
	var user sess
	err := session.Get(r, &user, h.sessionConfig())
	if err != nil && err != http.ErrNoCookie {
		h.deleteCookie(w)
		http.Error(w, "internal error", 500)
		return ctx, false
	}

	redirectURL := "https://" + r.Host + callbackPath

	conf := &oauth2.Config{
		ClientID:     h.ClientID,
		ClientSecret: h.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       h.Scopes,
		Endpoint:     Endpoint,
	}
	if conf.Scopes == nil {
		conf.Scopes = []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"}
	}
	if user.OAuthToken != nil {
		session.Set(w, user, h.sessionConfig()) // refresh the cookie
		ctx = context.WithValue(ctx, sessionKey, &Session{
			Client: conf.Client(ctx, user.OAuthToken),
		})
		return ctx, true
	}
	if r.URL.Path == callbackPath {
		if r.FormValue("state") != user.State {
			h.deleteCookie(w)
			grohl.Log(grohl.Data{"at": "loginOK", "what": "Mismatched state"})
			http.Error(w, "access forbidden", 401)
			return ctx, false
		}
		tok, err := conf.Exchange(ctx, r.FormValue("code"))
		if err != nil {
			h.deleteCookie(w)
			grohl.Log(grohl.Data{"at": "loginOK", "what": "Invalid credentials", "err": err})
			http.Error(w, "access forbidden", 401)
			return ctx, false
		}

		client := conf.Client(ctx, tok)
		if h.RequireDomain != "" && !domainAllowed(client, h.RequireDomain) {
			h.deleteCookie(w)
			http.Error(w, "access forbidden", 401)
			return ctx, false
		}

		session.Set(w, sess{OAuthToken: tok}, h.sessionConfig())
		http.Redirect(w, r, user.NextURL, http.StatusTemporaryRedirect)
		return ctx, false
	}

	u := *r.URL
	u.Scheme = "https"
	u.Host = r.Host
	state := newState()
	session.Set(w, sess{NextURL: u.String(), State: state}, h.sessionConfig())
	http.Redirect(w, r, conf.AuthCodeURL(state), http.StatusTemporaryRedirect)
	return ctx, false
}

// GoogleProfile stores information from the users google+ profile.
type googleProfile struct {
	ID          string `json:"id"`
	DisplayName string `json:"name"`
	FamilyName  string `json:"family_name"`
	GivenName   string `json:"given_name"`
	Email       string `json:"email"`
}

func domainAllowed(client *http.Client, domain string) bool {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v1/userinfo")
	if err != nil || resp.StatusCode != 200 {
		grohl.Log(grohl.Data{"at": "loginOK", "what": "Couldn't reach Google", "statuscode": resp.StatusCode, "err": err})
		return false
	}

	defer resp.Body.Close()

	gp := new(googleProfile)
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(gp); err != nil {
		grohl.Log(grohl.Data{"at": "loginOK", "what": "Failed to decode json", "err": err})
		return false
	}

	user := strings.Split(gp.Email, "@")
	if len(user) < 2 || user[1] != domain {
		grohl.Log(grohl.Data{"at": "loginOK", "what": "Invalid email", "email": gp.Email})
		return false
	}

	return true
}

func keys(s string) []*[32]byte {
	// e.g. faba0c08be7474a785b272c4f4154c998c0943b51e662637be11b1a0ecda43b3
	key, err := hex.DecodeString(os.Getenv("KEY"))
	if err != nil {
		grohl.Log(grohl.Data{"at": "keys", "what": "Invalid Key code", "err": err.Error()})
		os.Exit(1)
	}
	if len(key) != 32 {
		grohl.Log(grohl.Data{"at": "keys", "what": "Invalid Key length", "wanted": "32", "got": len(key)})
		os.Exit(1)
	}

	var key_array [32]byte
	copy(key_array[:], key)
	return []*[32]byte{&key_array}
}

func (h *Handler) sessionConfig() *session.Config {
	c := &session.Config{
		Name:   h.Name,
		Path:   h.Path,
		Domain: h.Domain,
		MaxAge: h.MaxAge,
		Keys:   keys(h.Key),
	}
	if c.Name == "" {
		c.Name = "googlegoauth"
	}
	return c
}

func (h *Handler) deleteCookie(w http.ResponseWriter) error {
	conf := h.sessionConfig()
	conf.MaxAge = -1 * time.Second
	return session.Set(w, sess{}, conf)
}

type sess struct {
	OAuthToken *oauth2.Token `json:",omitempty"`
	NextURL    string        `json:",omitempty"`
	State      string        `json:",omitempty"`
}

func newState() string {
	b := make([]byte, 10)
	rand.Read(b)
	return hex.EncodeToString(b)
}

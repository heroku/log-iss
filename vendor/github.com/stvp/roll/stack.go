package roll

import (
	"fmt"
	"hash/crc32"
	"os"
	"runtime"
	"strings"
)

var (
	knownFilePathPatterns []string = []string{
		"github.com/",
		"code.google.com/",
		"bitbucket.org/",
		"launchpad.net/",
	}
)

type stack []frame

type frame struct {
	Filename string `json:"filename"`
	Method   string `json:"method"`
	Line     int    `json:"lineno"`
}

func buildStack(skip int) stack {
	s := make(stack, 0)

	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		file = shortenFilePath(file)
		s = append(s, frame{file, functionName(pc), line})
	}

	return s
}

// Create a fingerprint that uniquely identify a given message. We use the full
// callstack, including file names. That ensure that there are no false
// duplicates but also means that after changing the code (adding/removing
// lines), the fingerprints will change. It's a trade-off.
func stackFingerprint(title string, s stack) string {
	hash := crc32.NewIEEE()
	fmt.Fprintf(hash, "%s", title)
	for _, frame := range s {
		fmt.Fprintf(hash, "%s%s%d", frame.Filename, frame.Method, frame.Line)
	}
	return fmt.Sprintf("%x", hash.Sum32())
}

// Remove un-needed information from the source file path. This makes them
// shorter in Rollbar UI as well as making them the same, regardless of the
// machine the code was compiled on.
//
// Examples:
//   /usr/local/go/src/pkg/runtime/proc.c -> pkg/runtime/proc.c
//   /home/foo/go/src/github.com/stvp/roll/rollbar.go -> stvp/roll/rollbar.go
func shortenFilePath(s string) string {
	idx := strings.Index(s, "/src/pkg/")
	if idx != -1 {
		return s[idx+5:]
	}
	for _, pattern := range knownFilePathPatterns {
		idx = strings.Index(s, pattern)
		if idx != -1 {
			return s[idx:]
		}
	}
	return s
}

func functionName(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "???"
	}
	name := fn.Name()
	end := strings.LastIndex(name, string(os.PathSeparator))
	return name[end+1 : len(name)]
}

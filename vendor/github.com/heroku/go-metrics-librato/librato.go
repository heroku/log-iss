package librato

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"time"

	"github.com/heroku/go-metrics"
)

// a regexp for extracting the unit from time.Duration.String
var unitRegexp = regexp.MustCompile(`[^\d]+$`)

// a helper that turns a time.Duration into librato display attributes for timer metrics
func translateTimerAttributes(d time.Duration) (attrs map[string]interface{}) {
	attrs = make(map[string]interface{})
	attrs[DisplayTransform] = fmt.Sprintf("x/%d", int64(d))
	attrs[DisplayUnitsShort] = string(unitRegexp.Find([]byte(d.String())))
	return
}

type Reporter struct {
	Email, Token    string
	Namespace       string
	Source          string
	Interval        time.Duration
	Registry        metrics.Registry
	Percentiles     []float64              // percentiles to report on histogram metrics
	TimerAttributes map[string]interface{} // units in which timers will be displayed
	intervalSec     int64
	resetCounters   bool
}

func NewReporter(r metrics.Registry, d time.Duration, e string, t string, n string, s string, p []float64, u time.Duration, resetCounters bool) *Reporter {
	return &Reporter{e, t, n, s, d, r, p, translateTimerAttributes(u), int64(d / time.Second), resetCounters}
}

func Librato(ctx context.Context, r metrics.Registry, d time.Duration, e, t, n, s string, p []float64, u time.Duration, resetCounters bool) {
	NewReporter(r, d, e, t, n, s, p, u, resetCounters).Run(ctx)
}

func (r *Reporter) Run(ctx context.Context) {
	ticker := time.NewTicker(r.Interval)
	metricsApi := &LibratoClient{r.Email, r.Token}

	for {
		select {
		case now := <-ticker.C:
			r.post(metricsApi, now)

		case <-ctx.Done():
			now := time.Now()
			r.post(metricsApi, now)
			log.Printf("go-metrics-librato: close received, cleaned up in %dms", time.Since(now)/time.Millisecond)
			return
		}
	}
}

func (r *Reporter) post(metricsApi *LibratoClient, now time.Time) error {
	metrics, err := r.BuildRequest(now, r.Registry, r.resetCounters)

	_, found := os.LookupEnv("DEBUG")
	if found {
		log.Printf("at=librato-body body=%+v", metrics)
	}

	if err != nil {
		log.Printf("ERROR constructing librato request body %s", err)
		return err
	}

	// NOTE: We *don't* want to prematurely cancel open HTTP Requests,
	// since we may be trying to flush before exit so we don't pass the
	// context to PostMetrics.
	if err = metricsApi.PostMetrics(metrics); err != nil {
		log.Printf("ERROR sending metrics to librato %s", err)
	}
	return err
}

// calculate sum of squares from data provided by metrics.Histogram
// see http://en.wikipedia.org/wiki/Standard_deviation#Rapid_calculation_methods
func sumSquares(s metrics.Sample) float64 {
	count := float64(s.Count())
	sumSquared := math.Pow(count*s.Mean(), 2)
	sumSquares := math.Pow(count*s.StdDev(), 2) + sumSquared/count
	if math.IsNaN(sumSquares) {
		return 0.0
	}
	return sumSquares
}
func sumSquaresTimer(t metrics.Timer) float64 {
	count := float64(t.Count())
	sumSquared := math.Pow(count*t.Mean(), 2)
	sumSquares := math.Pow(count*t.StdDev(), 2) + sumSquared/count
	if math.IsNaN(sumSquares) {
		return 0.0
	}
	return sumSquares
}

func (rep *Reporter) BuildRequest(now time.Time, r metrics.Registry, resetCounters bool) (snapshot Batch, err error) {
	snapshot = Batch{
		// coerce timestamps to a stepping fn so that they line up in Librato graphs
		MeasureTime: (now.Unix() / rep.intervalSec) * rep.intervalSec,
		Source:      rep.Source,
		Gauges:      make([]Measurement, 0),
		Counters:    make([]Measurement, 0),
	}
	histogramGaugeCount := 1 + len(rep.Percentiles)
	r.Each(func(name string, metric interface{}) {
		if rep.Namespace != "" {
			name = fmt.Sprintf("%s.%s", rep.Namespace, name)
		}
		measurement := Measurement{}
		measurement[Period] = rep.Interval.Seconds()
		switch m := metric.(type) {
		case metrics.Counter:
			if m.Count() > 0 {
				measurement[Name] = fmt.Sprintf("%s.%s", name, "count")
				measurement[Value] = float64(m.Count())
				measurement[Attributes] = map[string]interface{}{
					DisplayUnitsLong:  Operations,
					DisplayUnitsShort: OperationsShort,
					DisplayMin:        "0",
				}
				snapshot.Counters = append(snapshot.Counters, measurement)
			}
			if resetCounters {
				m.Clear()
			}
		case metrics.Gauge:
			measurement[Name] = name
			measurement[Value] = float64(m.Value())
			snapshot.Gauges = append(snapshot.Gauges, measurement)
		case metrics.GaugeFloat64:
			measurement[Name] = name
			measurement[Value] = float64(m.Value())
			snapshot.Gauges = append(snapshot.Gauges, measurement)
		case metrics.Histogram:
			if m.Count() > 0 {
				gauges := make([]Measurement, histogramGaugeCount, histogramGaugeCount)
				s := m.Sample()
				measurement[Name] = fmt.Sprintf("%s.%s", name, "hist")
				// For Librato, count must be the number of measurements in this sample. It will show sum/count as the mean.
				// Sample.Size() gives us this. Sample.Count() gives the total number of measurements ever recorded for the
				// life of the histogram, which means the Librato graph will trend toward 0 as more measurements are recored.
				measurement[Count] = uint64(s.Size())
				measurement[Max] = float64(s.Max())
				measurement[Min] = float64(s.Min())
				measurement[Sum] = float64(s.Sum())
				measurement[SumSquares] = sumSquares(s)
				gauges[0] = measurement
				for i, p := range rep.Percentiles {
					gauges[i+1] = Measurement{
						Name:   fmt.Sprintf("%s.%.2f", measurement[Name], p),
						Value:  s.Percentile(p),
						Period: measurement[Period],
					}
				}
				snapshot.Gauges = append(snapshot.Gauges, gauges...)
				s.Clear()
			}
		case metrics.Meter:
			measurement[Name] = name
			measurement[Value] = float64(m.Count())
			snapshot.Counters = append(snapshot.Counters, measurement)
			snapshot.Gauges = append(snapshot.Gauges,
				Measurement{
					Name:   fmt.Sprintf("%s.%s", name, "1min"),
					Value:  m.Rate1(),
					Period: int64(rep.Interval.Seconds()),
					Attributes: map[string]interface{}{
						DisplayUnitsLong:  Operations,
						DisplayUnitsShort: OperationsShort,
						DisplayMin:        "0",
					},
				},
				Measurement{
					Name:   fmt.Sprintf("%s.%s", name, "5min"),
					Value:  m.Rate5(),
					Period: int64(rep.Interval.Seconds()),
					Attributes: map[string]interface{}{
						DisplayUnitsLong:  Operations,
						DisplayUnitsShort: OperationsShort,
						DisplayMin:        "0",
					},
				},
				Measurement{
					Name:   fmt.Sprintf("%s.%s", name, "15min"),
					Value:  m.Rate15(),
					Period: int64(rep.Interval.Seconds()),
					Attributes: map[string]interface{}{
						DisplayUnitsLong:  Operations,
						DisplayUnitsShort: OperationsShort,
						DisplayMin:        "0",
					},
				},
			)
		case metrics.Timer:
			measurement[Name] = name
			measurement[Value] = float64(m.Count())
			snapshot.Counters = append(snapshot.Counters, measurement)
			if m.Count() > 0 {
				libratoName := fmt.Sprintf("%s.%s", name, "timer.mean")
				gauges := make([]Measurement, histogramGaugeCount, histogramGaugeCount)
				gauges[0] = Measurement{
					Name:       libratoName,
					Count:      uint64(m.Count()),
					Sum:        m.Mean() * float64(m.Count()),
					Max:        float64(m.Max()),
					Min:        float64(m.Min()),
					SumSquares: sumSquaresTimer(m),
					Period:     int64(rep.Interval.Seconds()),
					Attributes: rep.TimerAttributes,
				}
				for i, p := range rep.Percentiles {
					gauges[i+1] = Measurement{
						Name:       fmt.Sprintf("%s.timer.%2.0f", name, p*100),
						Value:      m.Percentile(p),
						Period:     int64(rep.Interval.Seconds()),
						Attributes: rep.TimerAttributes,
					}
				}
				snapshot.Gauges = append(snapshot.Gauges, gauges...)
				snapshot.Gauges = append(snapshot.Gauges,
					Measurement{
						Name:   fmt.Sprintf("%s.%s", name, "rate.1min"),
						Value:  m.Rate1(),
						Period: int64(rep.Interval.Seconds()),
						Attributes: map[string]interface{}{
							DisplayUnitsLong:  Operations,
							DisplayUnitsShort: OperationsShort,
							DisplayMin:        "0",
						},
					},
					Measurement{
						Name:   fmt.Sprintf("%s.%s", name, "rate.5min"),
						Value:  m.Rate5(),
						Period: int64(rep.Interval.Seconds()),
						Attributes: map[string]interface{}{
							DisplayUnitsLong:  Operations,
							DisplayUnitsShort: OperationsShort,
							DisplayMin:        "0",
						},
					},
					Measurement{
						Name:   fmt.Sprintf("%s.%s", name, "rate.15min"),
						Value:  m.Rate15(),
						Period: int64(rep.Interval.Seconds()),
						Attributes: map[string]interface{}{
							DisplayUnitsLong:  Operations,
							DisplayUnitsShort: OperationsShort,
							DisplayMin:        "0",
						},
					},
				)
			}
		}
	})
	return
}

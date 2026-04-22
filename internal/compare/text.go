package compare

import (
	"fmt"
	"io"
	"strings"
)

// RenderText writes a human-readable summary of the comparison to w. The
// layout is deliberately compact — top-N changes per source, followed by
// the recommendations outcome, calibration headline, and the honesty
// caveats. This format is the default for `sageo compare`.
func RenderText(w io.Writer, c *Comparison) error {
	if c == nil {
		return fmt.Errorf("compare: nil comparison")
	}

	b := &strings.Builder{}

	fmt.Fprintf(b, "Sageo Compare\n")
	fmt.Fprintf(b, "  from: %s (%s)\n", c.FromTime.Format("2006-01-02 15:04 MST"), shortRef(c.FromRef))
	fmt.Fprintf(b, "  to:   %s (%s)\n", c.ToTime.Format("2006-01-02 15:04 MST"), shortRef(c.ToRef))
	fmt.Fprintf(b, "  span: %s (%d days)\n\n", humanDuration(c.Duration), int(c.Duration.Hours()/24))

	// --- Recommendations outcome (headline) ---------------------------
	r := c.Recs
	fmt.Fprintf(b, "Recommendations\n")
	fmt.Fprintf(b, "  addressed:      %d\n", len(r.Addressed))
	fmt.Fprintf(b, "  still open:     %d\n", len(r.StillOpen))
	fmt.Fprintf(b, "  newly added:    %d\n", len(r.NewlyAdded))
	fmt.Fprintf(b, "  resolved (unflagged): %d\n", len(r.Resolved))
	if headline := calibrationHeadline(c); headline != "" {
		fmt.Fprintf(b, "  %s\n", headline)
	}
	fmt.Fprintln(b)

	if len(r.Addressed) > 0 {
		fmt.Fprintf(b, "Addressed changes (top %d)\n", minInt(len(r.Addressed), 10))
		for i, o := range r.Addressed {
			if i >= 10 {
				break
			}
			fmt.Fprintf(b, "  • [%s] %s\n", o.Recommendation.ChangeType, o.Recommendation.TargetURL)
			if o.Evidence != "" {
				fmt.Fprintf(b, "      evidence: %s\n", o.Evidence)
			}
			if o.ObservedLift != nil {
				fmt.Fprintf(b, "      observed: clicks %+d, impressions %+d, position %s (over %dd)\n",
					o.ObservedLift.ClicksDelta,
					o.ObservedLift.ImpressionsDelta,
					signedPos(o.ObservedLift.PositionDelta),
					o.ObservedLift.WindowDays)
				if o.Recommendation.ForecastedLift != nil {
					f := o.Recommendation.ForecastedLift
					fmt.Fprintf(b, "      forecast: %+d clicks/mo (range %d–%d)\n",
						f.Point(), f.Low(), f.High())
				}
				if o.ObservedLift.ConfidenceNote != "" {
					fmt.Fprintf(b, "      note: %s\n", o.ObservedLift.ConfidenceNote)
				}
			}
		}
		fmt.Fprintln(b)
	}

	// --- GSC ----------------------------------------------------------
	g := c.GSC
	if g.TotalQueriesFrom+g.TotalQueriesTo > 0 {
		fmt.Fprintf(b, "GSC\n")
		fmt.Fprintf(b, "  clicks Δ:       %+d\n", g.ClicksDelta)
		fmt.Fprintf(b, "  impressions Δ:  %+d\n", g.ImpressionsDelta)
		fmt.Fprintf(b, "  queries:        %d → %d (gained %d, lost %d)\n",
			g.TotalQueriesFrom, g.TotalQueriesTo, len(g.QueriesGained), len(g.QueriesLost))
		writeTopQueries(b, "  improved", g.PositionImproved, 5)
		writeTopQueries(b, "  declined", g.PositionDeclined, 5)
		fmt.Fprintln(b)
	}

	// --- PSI ----------------------------------------------------------
	if len(c.PSI.Changed) > 0 {
		fmt.Fprintf(b, "PSI (top %d by |ΔLCP|)\n", minInt(len(c.PSI.Changed), 5))
		for i, ch := range c.PSI.Changed {
			if i >= 5 {
				break
			}
			fmt.Fprintf(b, "  • %s [%s]: perf %.0f→%.0f, LCP %.0fms→%.0fms, CLS %.2f→%.2f\n",
				ch.URL, ch.Strategy,
				ch.FromPerformance*100, ch.ToPerformance*100,
				ch.FromLCP, ch.ToLCP, ch.FromCLS, ch.ToCLS)
			if ch.EnteredGoodBandLCP {
				fmt.Fprintf(b, "      LCP entered Core Web Vitals \"good\" band\n")
			}
			if ch.EnteredGoodBandCLS {
				fmt.Fprintf(b, "      CLS entered Core Web Vitals \"good\" band\n")
			}
		}
		fmt.Fprintln(b)
	}

	// --- SERP ---------------------------------------------------------
	if len(c.SERP.Changed) > 0 {
		fmt.Fprintf(b, "SERP (%d queries changed)\n", len(c.SERP.Changed))
		for i, ch := range c.SERP.Changed {
			if i >= 8 {
				break
			}
			fmt.Fprintf(b, "  • %q: pos %d→%d", ch.Query, ch.FromPosition, ch.ToPosition)
			if len(ch.FeaturesGained) > 0 {
				fmt.Fprintf(b, ", +features [%s]", strings.Join(ch.FeaturesGained, ","))
			}
			if len(ch.FeaturesLost) > 0 {
				fmt.Fprintf(b, ", -features [%s]", strings.Join(ch.FeaturesLost, ","))
			}
			if ch.AIOverviewGained {
				fmt.Fprintf(b, ", +AI overview")
			}
			if ch.AIOverviewLost {
				fmt.Fprintf(b, ", -AI overview")
			}
			fmt.Fprintln(b)
		}
		fmt.Fprintln(b)
	}

	// --- AEO ----------------------------------------------------------
	if len(c.AEO.Terms) > 0 {
		fmt.Fprintf(b, "AEO citations\n")
		for i, t := range c.AEO.Terms {
			if i >= 8 {
				break
			}
			fmt.Fprintf(b, "  • %s: %d → %d (%+d)", t.Term, t.CitationsFrom, t.CitationsTo, t.DeltaCitations)
			if len(t.PagesGained) > 0 {
				fmt.Fprintf(b, ", +%d pages", len(t.PagesGained))
			}
			if len(t.PagesLost) > 0 {
				fmt.Fprintf(b, ", -%d pages", len(t.PagesLost))
			}
			fmt.Fprintln(b)
		}
		fmt.Fprintln(b)
	}

	// --- Backlinks ----------------------------------------------------
	if c.Backlinks.Available {
		fmt.Fprintf(b, "Backlinks\n")
		fmt.Fprintf(b, "  backlinks Δ:        %+d\n", c.Backlinks.DeltaBacklinks)
		fmt.Fprintf(b, "  referring domains Δ: %+d (gained %d, lost %d in top list)\n",
			c.Backlinks.DeltaReferringDomains, c.Backlinks.DomainsGained, c.Backlinks.DomainsLost)
		fmt.Fprintf(b, "  broken Δ:           %+d\n", c.Backlinks.DeltaBroken)
		fmt.Fprintln(b)
	}

	// --- Audit --------------------------------------------------------
	a := c.Audit
	fmt.Fprintf(b, "Audit\n")
	fmt.Fprintf(b, "  score:     %.1f → %.1f (%+.1f)\n", a.FromScore, a.ToScore, a.DeltaScore)
	fmt.Fprintf(b, "  findings:  %d → %d (resolved %d, appeared %d)\n",
		a.FromFindingsCount, a.ToFindingsCount, a.FindingsResolved, a.FindingsAppeared)
	fmt.Fprintln(b)

	// --- Caveats ------------------------------------------------------
	fmt.Fprintf(b, "Caveats\n")
	for _, caveat := range c.Caveats {
		fmt.Fprintf(b, "  • %s\n", caveat)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func writeTopQueries(b *strings.Builder, label string, changes []GSCQueryChange, n int) {
	if len(changes) == 0 {
		return
	}
	fmt.Fprintf(b, "%s (top %d)\n", label, minInt(len(changes), n))
	for i, ch := range changes {
		if i >= n {
			break
		}
		fmt.Fprintf(b, "      %q: pos %.1f → %.1f (%s), clicks %+d\n",
			ch.Query, ch.FromPosition, ch.ToPosition,
			signedPos(ch.DeltaPosition), ch.DeltaClicks)
	}
}

// calibrationHeadline summarises whether the forecaster was systematically
// optimistic or pessimistic across the addressed recommendations that had
// both a prediction and a measurable outcome.
func calibrationHeadline(c *Comparison) string {
	var predicted, observed int
	pairs := 0
	for _, o := range c.Recs.Addressed {
		if o.ObservedLift == nil || o.Recommendation.ForecastedLift == nil {
			continue
		}
		predicted += o.Recommendation.ForecastedLift.Point()
		observed += o.ObservedLift.ClicksDelta
		pairs++
	}
	if pairs == 0 {
		return ""
	}
	if predicted == 0 {
		return fmt.Sprintf("calibration: %d measurable outcomes (predicted 0 clicks)", pairs)
	}
	ratio := float64(observed) / float64(predicted)
	bias := "on target"
	switch {
	case ratio < 0.7:
		bias = "optimistic"
	case ratio > 1.3:
		bias = "pessimistic"
	}
	return fmt.Sprintf("calibration: forecaster was %s — predicted %+d vs observed %+d across %d measurable outcomes",
		bias, predicted, observed, pairs)
}

func signedPos(d float64) string {
	if d < 0 {
		return fmt.Sprintf("%.1f improved", -d)
	}
	if d > 0 {
		return fmt.Sprintf("%.1f worse", d)
	}
	return "unchanged"
}

func humanDuration(d interface{ Hours() float64 }) string {
	hours := d.Hours()
	if hours < 24 {
		return fmt.Sprintf("%.1fh", hours)
	}
	days := hours / 24
	if days < 14 {
		return fmt.Sprintf("%.1fd", days)
	}
	return fmt.Sprintf("%.0fd", days)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func shortRef(r string) string {
	if len(r) > 20 {
		return r[:20] + "…"
	}
	return r
}

package compare

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

// RenderHTML writes a self-contained comparison report to w. The HTML is
// intentionally minimal and self-contained (inlined CSS, no external
// resources) so it can be emailed or committed alongside a snapshot. It
// leans on the same visual language as the main audit report but lives in
// its own template — compare output is narrative-shaped, not a full
// re-run of the audit sections.
func RenderHTML(w io.Writer, c *Comparison) error {
	if c == nil {
		return fmt.Errorf("compare: nil comparison")
	}
	tpl, err := template.New("compare").Funcs(template.FuncMap{
		"signedPos":   signedPos,
		"join":        strings.Join,
		"abs":         func(i int) int { return absInt(i) },
		"truncate":    truncate,
		"headline":    calibrationHeadline,
		"daysBetween": func(c *Comparison) int { return int(c.Duration.Hours() / 24) },
		"mul100":      func(f float64) float64 { return f * 100 },
	}).Parse(compareHTMLTemplate)
	if err != nil {
		return fmt.Errorf("compare: parse template: %w", err)
	}
	return tpl.Execute(w, c)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

const compareHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Sageo Compare — {{.FromTime.Format "2006-01-02"}} → {{.ToTime.Format "2006-01-02"}}</title>
<style>
  :root { --brand: #1E40AF; --ok: #047857; --bad: #b91c1c; --muted:#6b7280; }
  body { font: 14px/1.5 -apple-system, system-ui, sans-serif; margin: 2rem auto; max-width: 960px; color: #111827; padding: 0 1rem; }
  h1 { font-size: 1.6rem; margin-bottom: .25rem; }
  h2 { margin-top: 2rem; border-bottom: 1px solid #e5e7eb; padding-bottom: .25rem; }
  .meta { color: var(--muted); margin-bottom: 1.5rem; }
  .cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: .75rem; margin: 1rem 0; }
  .card { border: 1px solid #e5e7eb; border-radius: 8px; padding: .75rem 1rem; background: #f9fafb; }
  .card .n { font-size: 1.6rem; font-weight: 600; }
  .card .l { color: var(--muted); font-size: .8rem; text-transform: uppercase; letter-spacing: .03em; }
  table { border-collapse: collapse; width: 100%; margin: .5rem 0 1rem; }
  th, td { text-align: left; padding: .4rem .6rem; border-bottom: 1px solid #f3f4f6; font-size: .9rem; vertical-align: top; }
  th { background: #f9fafb; }
  .good { color: var(--ok); }
  .bad  { color: var(--bad); }
  .muted{ color: var(--muted); }
  .caveats { background: #fefce8; border: 1px solid #fde68a; border-radius: 8px; padding: 1rem; margin: 2rem 0; }
  .caveats h2 { border: 0; margin-top: 0; }
  .evidence { font-family: ui-monospace, Menlo, monospace; font-size: .82rem; color: var(--muted); }
  .headline { background: #eef2ff; border: 1px solid #c7d2fe; border-radius: 8px; padding: .75rem 1rem; margin: 1rem 0; }
</style>
</head>
<body>
  <h1>Compare</h1>
  <div class="meta">
    {{.FromTime.Format "2006-01-02 15:04 MST"}}
    &nbsp;→&nbsp;
    {{.ToTime.Format "2006-01-02 15:04 MST"}}
    &nbsp;·&nbsp;
    {{daysBetween .}} days
  </div>

  <div class="cards">
    <div class="card"><div class="n">{{len .Recs.Addressed}}</div><div class="l">Addressed</div></div>
    <div class="card"><div class="n">{{len .Recs.StillOpen}}</div><div class="l">Still open</div></div>
    <div class="card"><div class="n">{{len .Recs.NewlyAdded}}</div><div class="l">Newly added</div></div>
    <div class="card"><div class="n">{{len .Recs.Resolved}}</div><div class="l">Resolved (unflagged)</div></div>
  </div>

  {{with headline .}}<div class="headline">{{.}}</div>{{end}}

  {{if .Recs.Addressed}}
  <h2>Addressed recommendations</h2>
  <table>
    <thead><tr><th>Change</th><th>URL</th><th>Evidence</th><th>Observed</th><th>Forecast</th></tr></thead>
    <tbody>
    {{range .Recs.Addressed}}
      <tr>
        <td>{{.Recommendation.ChangeType}}</td>
        <td>{{truncate .Recommendation.TargetURL 60}}</td>
        <td class="evidence">{{.Evidence}}</td>
        <td>
          {{if .ObservedLift}}
            <span class="{{if gt .ObservedLift.ClicksDelta 0}}good{{else if lt .ObservedLift.ClicksDelta 0}}bad{{end}}">
              clicks {{.ObservedLift.ClicksDelta}}
            </span><br>
            <span class="muted">pos {{signedPos .ObservedLift.PositionDelta}}</span><br>
            {{if .ObservedLift.CitationsDelta}}<span class="muted">citations {{.ObservedLift.CitationsDelta}}</span><br>{{end}}
            <span class="muted">{{.ObservedLift.WindowDays}}d window</span>
          {{else}}<span class="muted">—</span>{{end}}
        </td>
        <td>
          {{with .Recommendation.ForecastedLift}}
            {{.Point}} clicks/mo<br>
            <span class="muted">({{.Low}}–{{.High}})</span>
          {{else}}<span class="muted">—</span>{{end}}
        </td>
      </tr>
    {{end}}
    </tbody>
  </table>
  {{end}}

  <h2>GSC</h2>
  <p>Clicks Δ <strong>{{.GSC.ClicksDelta}}</strong> · impressions Δ <strong>{{.GSC.ImpressionsDelta}}</strong> · queries {{.GSC.TotalQueriesFrom}} → {{.GSC.TotalQueriesTo}} ({{len .GSC.QueriesGained}} gained, {{len .GSC.QueriesLost}} lost)</p>
  {{if .GSC.PositionImproved}}
  <h3>Top improvements</h3>
  <table><thead><tr><th>Query</th><th>From</th><th>To</th><th>Δ pos</th><th>Δ clicks</th></tr></thead><tbody>
  {{range $i, $q := .GSC.PositionImproved}}{{if lt $i 10}}
    <tr><td>{{$q.Query}}</td><td>{{printf "%.1f" $q.FromPosition}}</td><td>{{printf "%.1f" $q.ToPosition}}</td><td class="good">{{signedPos $q.DeltaPosition}}</td><td>{{$q.DeltaClicks}}</td></tr>
  {{end}}{{end}}
  </tbody></table>
  {{end}}
  {{if .GSC.PositionDeclined}}
  <h3>Top declines</h3>
  <table><thead><tr><th>Query</th><th>From</th><th>To</th><th>Δ pos</th><th>Δ clicks</th></tr></thead><tbody>
  {{range $i, $q := .GSC.PositionDeclined}}{{if lt $i 10}}
    <tr><td>{{$q.Query}}</td><td>{{printf "%.1f" $q.FromPosition}}</td><td>{{printf "%.1f" $q.ToPosition}}</td><td class="bad">{{signedPos $q.DeltaPosition}}</td><td>{{$q.DeltaClicks}}</td></tr>
  {{end}}{{end}}
  </tbody></table>
  {{end}}

  {{if .PSI.Changed}}
  <h2>PSI</h2>
  <table><thead><tr><th>URL</th><th>Strategy</th><th>LCP</th><th>CLS</th><th>Performance</th></tr></thead><tbody>
  {{range $i, $p := .PSI.Changed}}{{if lt $i 10}}
    <tr>
      <td>{{truncate $p.URL 60}}</td><td>{{$p.Strategy}}</td>
      <td>{{printf "%.0f" $p.FromLCP}}ms → {{printf "%.0f" $p.ToLCP}}ms{{if $p.EnteredGoodBandLCP}} <span class="good">✓ good</span>{{end}}</td>
      <td>{{printf "%.2f" $p.FromCLS}} → {{printf "%.2f" $p.ToCLS}}{{if $p.EnteredGoodBandCLS}} <span class="good">✓ good</span>{{end}}</td>
      <td>{{printf "%.0f" (mul100 $p.FromPerformance)}} → {{printf "%.0f" (mul100 $p.ToPerformance)}}</td>
    </tr>
  {{end}}{{end}}
  </tbody></table>
  {{end}}

  {{if .AEO.Terms}}
  <h2>AEO citations</h2>
  <table><thead><tr><th>Term</th><th>From</th><th>To</th><th>Δ</th><th>Pages gained / lost</th></tr></thead><tbody>
  {{range .AEO.Terms}}
    <tr>
      <td>{{.Term}}</td><td>{{.CitationsFrom}}</td><td>{{.CitationsTo}}</td>
      <td class="{{if gt .DeltaCitations 0}}good{{else if lt .DeltaCitations 0}}bad{{end}}">{{.DeltaCitations}}</td>
      <td class="muted">+{{len .PagesGained}} / -{{len .PagesLost}}</td>
    </tr>
  {{end}}
  </tbody></table>
  {{end}}

  {{if .Backlinks.Available}}
  <h2>Backlinks</h2>
  <p>Backlinks Δ <strong>{{.Backlinks.DeltaBacklinks}}</strong> · referring domains Δ <strong>{{.Backlinks.DeltaReferringDomains}}</strong> · broken Δ <strong>{{.Backlinks.DeltaBroken}}</strong></p>
  {{end}}

  <h2>Audit</h2>
  <p>Score {{printf "%.1f" .Audit.FromScore}} → {{printf "%.1f" .Audit.ToScore}} · findings {{.Audit.FromFindingsCount}} → {{.Audit.ToFindingsCount}} (resolved {{.Audit.FindingsResolved}}, appeared {{.Audit.FindingsAppeared}})</p>

  <div class="caveats">
    <h2>Reading this report</h2>
    <ul>
      {{range .Caveats}}<li>{{.}}</li>{{end}}
    </ul>
  </div>
</body>
</html>
`

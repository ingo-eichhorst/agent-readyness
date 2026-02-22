// report generates benchmark/report.html from benchmark/golden/**/*.json and benchmark/benchmark.yaml.
//
// Usage:
//
//	go run ./benchmark/report
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// ---- Data types ----

type yamlRepo struct {
	Name          string  `yaml:"name"`
	Language      string  `yaml:"language"`
	URL           string  `yaml:"url"`
	Commit        string  `yaml:"commit"`
	Size          string  `yaml:"size"`
	ExpectedQuality string `yaml:"expected_quality"`
}

type yamlConfig struct {
	Repos []yamlRepo `yaml:"repos"`
}

type goldenCategory struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

type goldenFile struct {
	CompositeScore float64          `json:"composite_score"`
	Tier           string           `json:"tier"`
	Categories     []goldenCategory `json:"categories"`
}

// RepoData is passed to the HTML template.
type RepoData struct {
	Name     string
	Lang     string
	URL      string
	Commit   string
	Score    float64
	Tier     string
	Cats     map[string]float64 // -1 = N/A
}

type TemplateData struct {
	GeneratedAt string
	Repos       []RepoData
	Stats       Stats
}

type Stats struct {
	Total        int
	AgentReady   int
	AgentAssisted int
	AgentLimited int
	AgentBlocked int
	AvgGo        float64
	AvgPython    float64
	AvgTS        float64
	RangeGo      string
	RangePython  string
	RangeTS      string
}

func main() {
	// Locate repo root (script is in benchmark/report/, run from anywhere)
	root, err := findRoot()
	if err != nil {
		fatal("cannot find repo root: %v", err)
	}

	// Load YAML config
	yamlPath := filepath.Join(root, "benchmark", "benchmark.yaml")
	cfg, err := loadYAML(yamlPath)
	if err != nil {
		fatal("load benchmark.yaml: %v", err)
	}

	// Build lookup: name -> yamlRepo
	byName := make(map[string]yamlRepo, len(cfg.Repos))
	for _, r := range cfg.Repos {
		byName[r.Name] = r
	}

	// Load all golden files
	goldenDir := filepath.Join(root, "benchmark", "golden")
	repos, err := loadGoldens(goldenDir, byName)
	if err != nil {
		fatal("load goldens: %v", err)
	}

	if len(repos) == 0 {
		fatal("no golden files found in %s — run the benchmark with -update first", goldenDir)
	}

	// Compute stats
	stats := computeStats(repos)

	// Render HTML
	outPath := filepath.Join(root, "benchmark", "report.html")
	if err := render(outPath, TemplateData{
		GeneratedAt: time.Now().UTC().Format("2006-01-02 15:04 UTC"),
		Repos:       repos,
		Stats:       stats,
	}); err != nil {
		fatal("render: %v", err)
	}

	fmt.Printf("report written to %s\n", outPath)
}

// findRoot walks up from the executable/working directory to find go.mod.
func findRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found")
}

func loadYAML(path string) (*yamlConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg yamlConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func loadGoldens(dir string, meta map[string]yamlRepo) ([]RepoData, error) {
	var repos []RepoData

	// Walk golden/go/, golden/python/, golden/typescript/
	langs := []string{"go", "python", "typescript"}
	for _, lang := range langs {
		langDir := filepath.Join(dir, lang)
		entries, err := os.ReadDir(langDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", langDir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".json")
			data, err := os.ReadFile(filepath.Join(langDir, e.Name()))
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", e.Name(), err)
			}
			var gf goldenFile
			if err := json.Unmarshal(data, &gf); err != nil {
				return nil, fmt.Errorf("parse %s: %w", e.Name(), err)
			}
			cats := make(map[string]float64)
			for _, c := range gf.Categories {
				cats[c.Name] = c.Score
			}
			m := meta[name]
			repos = append(repos, RepoData{
				Name:   name,
				Lang:   lang,
				URL:    m.URL,
				Commit: m.Commit,
				Score:  gf.CompositeScore,
				Tier:   gf.Tier,
				Cats:   cats,
			})
		}
	}

	// Sort to match benchmark.yaml order (preserve language grouping)
	langOrder := map[string]int{"go": 0, "python": 1, "typescript": 2}
	sort.SliceStable(repos, func(i, j int) bool {
		li, lj := langOrder[repos[i].Lang], langOrder[repos[j].Lang]
		if li != lj {
			return li < lj
		}
		return repos[i].Name < repos[j].Name
	})

	return repos, nil
}

func computeStats(repos []RepoData) Stats {
	s := Stats{Total: len(repos)}
	type langStat struct{ sum float64; count, n int; min, max float64 }
	byLang := map[string]*langStat{
		"go": {min: math.MaxFloat64}, "python": {min: math.MaxFloat64}, "typescript": {min: math.MaxFloat64},
	}
	for _, r := range repos {
		switch r.Tier {
		case "Agent-Ready":
			s.AgentReady++
		case "Agent-Assisted":
			s.AgentAssisted++
		case "Agent-Limited":
			s.AgentLimited++
		case "Agent-Blocked":
			s.AgentBlocked++
		}
		ls := byLang[r.Lang]
		if ls != nil {
			ls.sum += r.Score
			ls.count++
			if r.Score < ls.min {
				ls.min = r.Score
			}
			if r.Score > ls.max {
				ls.max = r.Score
			}
		}
	}
	avg := func(ls *langStat) float64 {
		if ls.count == 0 {
			return 0
		}
		return ls.sum / float64(ls.count)
	}
	rng := func(ls *langStat) string {
		if ls.count == 0 {
			return "—"
		}
		return fmt.Sprintf("%.2f–%.2f", ls.min, ls.max)
	}
	s.AvgGo = avg(byLang["go"])
	s.AvgPython = avg(byLang["python"])
	s.AvgTS = avg(byLang["typescript"])
	s.RangeGo = rng(byLang["go"])
	s.RangePython = rng(byLang["python"])
	s.RangeTS = rng(byLang["typescript"])
	return s
}

func render(outPath string, data TemplateData) error {
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"json": func(v any) (string, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
		"f2": func(v float64) string { return fmt.Sprintf("%.2f", v) },
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "report: "+format+"\n", args...)
	os.Exit(1)
}

// ---- HTML template ----
// The JS receives a JSON blob of repos and builds all the UI client-side,
// keeping the template simple and the HTML self-contained.

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ARS Benchmark Report — {{.Stats.Total}} Open-Source Repos</title>
<style>
  :root {
    --bg: #0f1117;
    --surface: #1a1d27;
    --surface2: #22263a;
    --border: #2e3348;
    --text: #e2e8f0;
    --muted: #8892a4;
    --ready: #22c55e;
    --assisted: #3b82f6;
    --limited: #f59e0b;
    --blocked: #ef4444;
    --c1: #a78bfa;
    --c2: #38bdf8;
    --c3: #fb923c;
    --c4: #f472b6;
    --c5: #4ade80;
    --c6: #facc15;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { background: var(--bg); color: var(--text); font-family: 'Inter', system-ui, sans-serif; font-size: 14px; line-height: 1.6; }
  a { color: var(--assisted); text-decoration: none; }
  .page { max-width: 1280px; margin: 0 auto; padding: 2rem 1.5rem 4rem; }
  header { margin-bottom: 2.5rem; }
  header h1 { font-size: 1.75rem; font-weight: 700; letter-spacing: -0.02em; }
  header p { color: var(--muted); margin-top: 0.4rem; }
  .meta { display: flex; gap: 1.5rem; margin-top: 1rem; flex-wrap: wrap; }
  .meta-item { display: flex; align-items: center; gap: 0.4rem; font-size: 0.8rem; color: var(--muted); }
  .dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
  .dot-ready    { background: var(--ready); }
  .dot-assisted { background: var(--assisted); }
  .dot-limited  { background: var(--limited); }
  .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 1rem; margin-bottom: 2.5rem; }
  .stat-card { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 1rem 1.25rem; }
  .stat-card .label { font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.08em; color: var(--muted); }
  .stat-card .value { font-size: 1.8rem; font-weight: 700; margin-top: 0.2rem; }
  .stat-card .sub   { font-size: 0.75rem; color: var(--muted); margin-top: 0.1rem; }
  .section-title { font-size: 1rem; font-weight: 600; margin-bottom: 1rem; color: var(--text); border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; }
  .chart-section { margin-bottom: 2.5rem; }
  .bar-chart { display: flex; flex-direction: column; gap: 0.45rem; }
  .bar-row { display: grid; grid-template-columns: 130px 1fr 50px; align-items: center; gap: 0.75rem; }
  .bar-label { font-size: 0.78rem; color: var(--text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; text-align: right; }
  .bar-track { background: var(--surface2); border-radius: 4px; height: 18px; overflow: hidden; }
  .bar-fill { height: 100%; border-radius: 4px; }
  .bar-score { font-size: 0.75rem; color: var(--muted); font-variant-numeric: tabular-nums; }
  .lang-label { font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.1em; font-weight: 600; grid-column: 1/-1; margin-top: 0.5rem; }
  .lang-sep { height: 1px; background: var(--border); margin: 0.5rem 0; grid-column: 1/-1; }
  .heatmap-section { margin-bottom: 2.5rem; overflow-x: auto; }
  table.heatmap { border-collapse: collapse; width: 100%; min-width: 700px; }
  table.heatmap th { font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted); padding: 0.4rem 0.6rem; text-align: center; border-bottom: 1px solid var(--border); white-space: nowrap; }
  table.heatmap th.name-col { text-align: left; }
  table.heatmap td { padding: 0.35rem 0.6rem; text-align: center; font-size: 0.78rem; border-bottom: 1px solid var(--border); font-variant-numeric: tabular-nums; }
  table.heatmap td.name-col { text-align: left; color: var(--text); white-space: nowrap; }
  table.heatmap td.lang-tag { font-size: 0.65rem; text-transform: uppercase; letter-spacing: 0.08em; }
  table.heatmap tr:hover td { background: var(--surface2); }
  .tier-badge { display: inline-block; padding: 2px 7px; border-radius: 4px; font-size: 0.65rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.06em; }
  .tier-ready    { background: rgba(34,197,94,0.15);  color: var(--ready); }
  .tier-assisted { background: rgba(59,130,246,0.15); color: var(--assisted); }
  .tier-limited  { background: rgba(245,158,11,0.15); color: var(--limited); }
  .tier-blocked  { background: rgba(239,68,68,0.15);  color: var(--blocked); }
  .radar-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 1.25rem; margin-bottom: 2.5rem; }
  .radar-card { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 1.25rem; }
  .radar-card h3 { font-size: 0.85rem; font-weight: 600; margin-bottom: 0.25rem; display: flex; align-items: center; justify-content: space-between; }
  .radar-card .composite { font-size: 1.4rem; font-weight: 700; }
  .radar-card .lang-pill { font-size: 0.62rem; text-transform: uppercase; letter-spacing: 0.08em; padding: 2px 6px; border-radius: 3px; background: var(--surface2); color: var(--muted); }
  .cat-bars { display: flex; flex-direction: column; gap: 5px; margin-top: 0.75rem; }
  .cat-row { display: grid; grid-template-columns: 22px 1fr 32px; gap: 6px; align-items: center; }
  .cat-name { font-size: 0.68rem; color: var(--muted); font-weight: 600; }
  .cat-track { background: var(--surface2); border-radius: 3px; height: 8px; overflow: hidden; }
  .cat-fill { height: 100%; border-radius: 3px; }
  .cat-val { font-size: 0.65rem; color: var(--muted); text-align: right; font-variant-numeric: tabular-nums; }
  .na { color: var(--border); font-style: italic; }
  .insights { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 1rem; margin-bottom: 2.5rem; }
  .insight-card { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 1.25rem; }
  .insight-card h3 { font-size: 0.78rem; text-transform: uppercase; letter-spacing: 0.08em; color: var(--muted); margin-bottom: 0.75rem; }
  .insight-list { display: flex; flex-direction: column; gap: 0.5rem; }
  .insight-row { display: flex; justify-content: space-between; align-items: center; font-size: 0.8rem; }
  .insight-row .repo-name { color: var(--text); }
  .insight-row .score-val { font-weight: 600; font-variant-numeric: tabular-nums; }
  footer { text-align: center; color: var(--muted); font-size: 0.75rem; margin-top: 2rem; }
</style>
</head>
<body>
<div class="page">

<header>
  <h1>ARS Benchmark Report</h1>
  <p>{{.Stats.Total}} open-source repositories &middot; Go, Python, TypeScript</p>
  <div class="meta">
    <div class="meta-item"><span class="dot dot-ready"></span> Agent-Ready (&ge;8.0)</div>
    <div class="meta-item"><span class="dot dot-assisted"></span> Agent-Assisted (6.0&ndash;7.9)</div>
    <div class="meta-item"><span class="dot dot-limited"></span> Agent-Limited (4.0&ndash;5.9)</div>
    <div class="meta-item" style="color:#8892a4">C7 requires LLM &middot; shown as N/A</div>
  </div>
</header>

<div class="stats">
  <div class="stat-card">
    <div class="label">Repos Scanned</div>
    <div class="value">{{.Stats.Total}}</div>
    <div class="sub">10 per language</div>
  </div>
  <div class="stat-card">
    <div class="label">Agent-Ready</div>
    <div class="value" style="color:var(--ready)">{{.Stats.AgentReady}}</div>
    <div class="sub">score &ge; 8.0</div>
  </div>
  <div class="stat-card">
    <div class="label">Agent-Assisted</div>
    <div class="value" style="color:var(--assisted)">{{.Stats.AgentAssisted}}</div>
    <div class="sub">score 6.0&ndash;7.9</div>
  </div>
  <div class="stat-card">
    <div class="label">Agent-Limited</div>
    <div class="value" style="color:var(--limited)">{{.Stats.AgentLimited}}</div>
    <div class="sub">score 4.0&ndash;5.9</div>
  </div>
  <div class="stat-card">
    <div class="label">Avg Score (Go)</div>
    <div class="value">{{f2 .Stats.AvgGo}}</div>
    <div class="sub">range {{.Stats.RangeGo}}</div>
  </div>
  <div class="stat-card">
    <div class="label">Avg Score (Python)</div>
    <div class="value">{{f2 .Stats.AvgPython}}</div>
    <div class="sub">range {{.Stats.RangePython}}</div>
  </div>
  <div class="stat-card">
    <div class="label">Avg Score (TS)</div>
    <div class="value">{{f2 .Stats.AvgTS}}</div>
    <div class="sub">range {{.Stats.RangeTS}}</div>
  </div>
</div>

<div class="chart-section">
  <div class="section-title">Composite Score &mdash; All {{.Stats.Total}} Repos (ranked)</div>
  <div class="bar-chart" id="barChart"></div>
</div>

<div class="heatmap-section">
  <div class="section-title">Category Scores Heatmap</div>
  <table class="heatmap">
    <thead>
      <tr>
        <th class="name-col">Repo</th>
        <th>Lang</th>
        <th>Score</th>
        <th>Tier</th>
        <th>C1<br><small>Code Health</small></th>
        <th>C2<br><small>Semantics</small></th>
        <th>C3<br><small>Architecture</small></th>
        <th>C4<br><small>Docs</small></th>
        <th>C5<br><small>Temporal</small></th>
        <th>C6<br><small>Testing</small></th>
      </tr>
    </thead>
    <tbody id="heatmapBody"></tbody>
  </table>
</div>

<div class="section-title">Per-Repo Category Breakdown</div>
<div class="radar-grid" id="radarGrid"></div>

<div class="section-title">Notable Findings</div>
<div class="insights">
  <div class="insight-card">
    <h3>Top Scorers</h3>
    <div class="insight-list" id="topScorers"></div>
  </div>
  <div class="insight-card">
    <h3>Lowest Scorers</h3>
    <div class="insight-list" id="lowScorers"></div>
  </div>
  <div class="insight-card">
    <h3>Best C2 Semantics</h3>
    <div class="insight-list" id="bestC2"></div>
  </div>
  <div class="insight-card">
    <h3>Weakest C4 Documentation</h3>
    <div class="insight-list" id="weakC4"></div>
  </div>
</div>

<footer>Generated by ARS benchmark suite &middot; {{.GeneratedAt}}</footer>
</div>

<script>
const repos = [{{range $i, $r := .Repos}}{{if $i}},{{end}}
  {name:{{json $r.Name}},lang:{{json $r.Lang}},url:{{json $r.URL}},commit:{{json $r.Commit}},score:{{f2 $r.Score}},tier:{{json $r.Tier}},cats:{C1:{{index $r.Cats "C1"}},C2:{{index $r.Cats "C2"}},C3:{{index $r.Cats "C3"}},C4:{{index $r.Cats "C4"}},C5:{{index $r.Cats "C5"}},C6:{{index $r.Cats "C6"}}}}{{end}}];

// Normalize: score strings → floats, -1 → null
repos.forEach(r => {
  r.score = parseFloat(r.score);
  for (const k in r.cats) {
    const v = parseFloat(r.cats[k]);
    r.cats[k] = (isNaN(v) || v < 0) ? null : v;
  }
});

const tierColor = {
  "Agent-Ready":    "var(--ready)",
  "Agent-Assisted": "var(--assisted)",
  "Agent-Limited":  "var(--limited)",
  "Agent-Blocked":  "var(--blocked)",
};
const tierClass = {
  "Agent-Ready":    "tier-ready",
  "Agent-Assisted": "tier-assisted",
  "Agent-Limited":  "tier-limited",
  "Agent-Blocked":  "tier-blocked",
};
const catColors = {C1:"var(--c1)",C2:"var(--c2)",C3:"var(--c3)",C4:"var(--c4)",C5:"var(--c5)",C6:"var(--c6)"};
const catLabels = {C1:"Code Health",C2:"Semantics",C3:"Architecture",C4:"Docs",C5:"Temporal",C6:"Testing"};
const langColors = {go:"#00ADD8", python:"#3572A5", typescript:"#2b7489"};

function heatColor(v) {
  if (v === null) return "transparent";
  const t = Math.max(0, Math.min(1, (v - 1) / 9));
  if (t < 0.5) {
    const r = Math.round(239 + (245-239)*t*2);
    const g = Math.round(68  + (158-68)*t*2);
    const b = Math.round(68  + (11-68)*t*2);
    return 'rgba('+r+','+g+','+b+',0.35)';
  } else {
    const tt = (t-0.5)*2;
    const r = Math.round(245 + (34-245)*tt);
    const g = Math.round(158 + (197-158)*tt);
    const b = Math.round(11  + (94-11)*tt);
    return 'rgba('+r+','+g+','+b+',0.35)';
  }
}

// Bar chart (ranked within language groups)
const byLang = {};
repos.forEach(r => { (byLang[r.lang] = byLang[r.lang] || []).push(r); });
const langOrder = ["go","python","typescript"];
const barChart = document.getElementById('barChart');
let firstLang = true;
langOrder.forEach(lang => {
  const group = (byLang[lang] || []).slice().sort((a,b) => b.score - a.score);
  if (!group.length) return;
  if (!firstLang) {
    const sep = document.createElement('div');
    sep.className = 'bar-row';
    sep.innerHTML = '<div class="lang-sep"></div>';
    barChart.appendChild(sep);
  }
  firstLang = false;
  const lbl = document.createElement('div');
  lbl.className = 'bar-row';
  lbl.innerHTML = '<div class="lang-label" style="color:'+langColors[lang]+'">'+lang.toUpperCase()+'</div>';
  barChart.appendChild(lbl);
  group.forEach(r => {
    const pct = (r.score / 10 * 100).toFixed(1);
    const row = document.createElement('div');
    row.className = 'bar-row';
    row.innerHTML =
      '<div class="bar-label">'+r.name+'</div>'+
      '<div class="bar-track"><div class="bar-fill" style="width:'+pct+'%;background:'+tierColor[r.tier]+'"></div></div>'+
      '<div class="bar-score">'+r.score+'</div>';
    barChart.appendChild(row);
  });
});

// Heatmap table
const tbody = document.getElementById('heatmapBody');
repos.forEach(r => {
  const tr = document.createElement('tr');
  const cats = ['C1','C2','C3','C4','C5','C6'];
  let catCells = cats.map(c => {
    const v = r.cats[c];
    const bg = heatColor(v);
    const txt = v !== null ? v.toFixed(1) : '<span class="na">N/A</span>';
    return '<td style="background:'+bg+'">'+txt+'</td>';
  }).join('');
  const nameLink = r.url ? '<a href="'+r.url+'" target="_blank">'+r.name+'</a>' : r.name;
  tr.innerHTML =
    '<td class="name-col"><strong>'+nameLink+'</strong></td>'+
    '<td class="lang-tag" style="color:'+langColors[r.lang]+'">'+r.lang+'</td>'+
    '<td style="font-weight:600;color:'+tierColor[r.tier]+'">'+r.score+'</td>'+
    '<td><span class="tier-badge '+tierClass[r.tier]+'">'+r.tier.replace('Agent-','')+'</span></td>'+
    catCells;
  tbody.appendChild(tr);
});

// Radar cards
const grid = document.getElementById('radarGrid');
repos.forEach(r => {
  const card = document.createElement('div');
  card.className = 'radar-card';
  const cats = ['C1','C2','C3','C4','C5','C6'];
  const catRows = cats.map(c => {
    const v = r.cats[c];
    const pct = v !== null ? (v / 10 * 100).toFixed(0) : 0;
    const val = v !== null ? v.toFixed(1) : '<span class="na">N/A</span>';
    return '<div class="cat-row">'+
      '<div class="cat-name">'+c+'</div>'+
      '<div class="cat-track"><div class="cat-fill" style="width:'+pct+'%;background:'+catColors[c]+'"></div></div>'+
      '<div class="cat-val">'+val+'</div>'+
      '</div>';
  }).join('');
  const nameLink = r.url ? '<a href="'+r.url+'" target="_blank">'+r.name+'</a>' : r.name;
  card.innerHTML =
    '<h3><span>'+nameLink+'</span><span class="lang-pill" style="color:'+langColors[r.lang]+'">'+r.lang+'</span></h3>'+
    '<div style="display:flex;align-items:baseline;gap:0.5rem;margin-bottom:0.5rem">'+
      '<span class="composite" style="color:'+tierColor[r.tier]+'">'+r.score+'</span>'+
      '<span class="tier-badge '+tierClass[r.tier]+'">'+r.tier.replace('Agent-','')+'</span>'+
    '</div>'+
    '<div class="cat-bars">'+catRows+'</div>';
  grid.appendChild(card);
});

// Insights
function fillInsight(id, items) {
  const el = document.getElementById(id);
  items.forEach(function(item) {
    const row = document.createElement('div');
    row.className = 'insight-row';
    row.innerHTML = '<span class="repo-name">'+item.name+' <span style="font-size:0.65rem;color:'+langColors[item.lang]+'">'+item.lang+'</span></span><span class="score-val">'+item.val+'</span>';
    el.appendChild(row);
  });
}

const byScore = repos.slice().sort((a,b) => b.score - a.score);
fillInsight('topScorers', byScore.slice(0,5).map(r => ({name:r.name, lang:r.lang, val:r.score})));
fillInsight('lowScorers', byScore.slice(-5).reverse().map(r => ({name:r.name, lang:r.lang, val:r.score})));

const byC2 = repos.filter(r => r.cats.C2 !== null).sort((a,b) => b.cats.C2 - a.cats.C2);
fillInsight('bestC2', byC2.slice(0,5).map(r => ({name:r.name, lang:r.lang, val:r.cats.C2.toFixed(2)})));

const byC4 = repos.filter(r => r.cats.C4 !== null).sort((a,b) => a.cats.C4 - b.cats.C4);
fillInsight('weakC4', byC4.slice(0,5).map(r => ({name:r.name, lang:r.lang, val:r.cats.C4.toFixed(2)})));
</script>
</body>
</html>`

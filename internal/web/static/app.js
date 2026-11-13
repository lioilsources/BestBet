// BestBet hint app (plan 03). Plain JS over /api; no build step.
"use strict";

const SEL = ["home", "draw", "away"];
const SEL_LABEL = { home: "1", draw: "X", away: "2" };
const BOOKS = { "betano.cz": "Betano", "merkurxtip.cz": "MerkurXtip", sharp: "Ostrý trh", pinnacle: "Ostrý trh" };
const COLORS = { sharp: "var(--ink)", "betano.cz": "var(--s-home)", "merkurxtip.cz": "var(--s-away)" };
const view = document.getElementById("view");

const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const pct = (p, d = 1) => (p == null ? "–" : (100 * p).toFixed(d).replace(".", ",") + " %");
const pp = (x) => (x >= 0 ? "+" : "−") + Math.abs(100 * x).toFixed(1).replace(".", ",");
const num = (x, d = 2) => (x == null ? "–" : x.toFixed(d).replace(".", ","));
const book = (b) => BOOKS[b] || b;
const fmtTime = (t) => new Date(t).toLocaleString("cs-CZ", { weekday: "short", day: "numeric", month: "numeric", hour: "2-digit", minute: "2-digit" });
const ago = (t) => {
  const m = Math.round((Date.now() - new Date(t)) / 60000);
  if (m < 60) return `před ${m} min`;
  if (m < 48 * 60) return `před ${Math.round(m / 60)} h`;
  return `před ${Math.round(m / 1440)} dny`;
};

async function api(path) {
  const r = await fetch(path);
  if (!r.ok) throw new Error(`${path}: ${r.status}`);
  return r.json();
}

function setTab(tab) {
  document.querySelectorAll("nav a").forEach((a) => a.classList.toggle("on", a.dataset.tab === tab));
}

async function route() {
  const h = location.hash.replace(/^#/, "") || "/";
  try {
    if (h.startsWith("/game/")) { setTab("board"); await renderGame(h.split("/")[2]); }
    else if (h === "/eval") { setTab("eval"); await renderEval(); }
    else if (h === "/data") { setTab("data"); await renderData(); }
    else { setTab("board"); await renderBoard(); }
  } catch (e) {
    view.innerHTML = `<p class="err">Chyba: ${esc(e.message)}</p>`;
  }
  window.scrollTo(0, 0);
}
window.addEventListener("hashchange", route);

// ---------- Board ----------

function hintBox(g) {
  const h = g.hint, r = g.recorded;
  if (!h && !r) {
    return g.sharp ? `<div class="nohint">Bez hintu: CZ knihy nejsou proti ostrému trhu pozadu o ≥ 3 p. b.</div>`
                   : `<div class="nohint">Bez ostrého trhu, hint nelze spočítat.</div>`;
  }
  let s = "";
  if (h) {
    s += `<div class="hint"><b>Hint: ${SEL_LABEL[h.selection]} u ${esc(book(h.book))} @ ${num(h.price)}</b><span class="tag">ve zkoušce</span><br>
      Ostrý trh dává ${pct(h.p_sharp)}, kniha ${pct(h.p_soft)} (hrana ${pp(h.edge)} p. b.), EV ${pp(h.ev)} %.</div>`;
  }
  if (r) {
    s += `<div class="nohint">V deníku: ${SEL_LABEL[r.selection]} u ${esc(book(r.book))} @ ${num(r.price)}, zapsáno ${fmtTime(r.recorded_at)}.</div>`;
  }
  return s;
}

function grid(g) {
  const rows = [];
  if (g.sharp) {
    rows.push(`<tr><td>Ostrý trh</td>${SEL.map((s, i) => {
      let mv = "";
      if (g.sharp_open) {
        const d = g.sharp[i] - g.sharp_open[i];
        if (Math.abs(d) >= 0.005) mv = `<span class="mv ${d > 0 ? "up" : "down"}">${d > 0 ? "▲" : "▼"}${Math.abs(100 * d).toFixed(1).replace(".", ",")}</span>`;
      }
      return `<td>${pct(g.sharp[i], 0)}${mv}</td>`;
    }).join("")}</tr>`);
  }
  if (g.p_model) rows.push(`<tr><td>Náš model</td>${g.p_model.map((p) => `<td>${pct(p, 0)}</td>`).join("")}</tr>`);
  if (g.best && Object.keys(g.best).length) {
    rows.push(`<tr><td>Nejlepší kurz</td>${SEL.map((s) => {
      const b = g.best[s];
      return `<td>${b ? `${num(b.price)}<span class="book">${esc(book(b.book))}</span>` : "–"}</td>`;
    }).join("")}</tr>`);
  }
  if (!rows.length) return `<p class="muted">Zatím bez kurzů a predikce.</p>`;
  return `<table class="g"><tr><th></th><th>1</th><th>X</th><th>2</th></tr>${rows.join("")}</table>`;
}

const LINEUP_STATUS = { estimate: "odhad z posledního zápasu", available: "dostupní hráči (NHL, asi 90 min před zápasem)", confirmed: "potvrzená sestava", unknown: "málo zápasů v sezoně" };

function lineupTag(g) {
  if (!g.lineup) return "";
  const t = [["home", g.home_name], ["away", g.away_name]]
    .filter(([s]) => g.lineup[s].key_missing > 0)
    .map(([s, n]) => `${esc(n)}: chybí ${g.lineup[s].key_missing} klíčov${g.lineup[s].key_missing === 1 ? "ý" : "í"}`);
  return t.length ? `<div class="nohint">${t.join(" · ")}</div>` : "";
}

function lineupTeam(name, t) {
  const extra = t.status === "available" && t.listed > 20 ? `, ${t.listed - 20} z nich ještě vypadne` : "";
  const status = `${LINEUP_STATUS[t.status] || t.status}${t.confirmed_at ? `, ${fmtTime(t.confirmed_at)}` : ""}${extra}`;
  if (t.status === "unknown") return `<div class="card"><div class="teams">${esc(name)}</div><div class="sub">${status}</div></div>`;
  const row = (p, extra) => `<tr><td class="l">${esc(p.name || "#" + p.player_id)}${p.key ? ' <span class="tag">klíčový</span>' : ""}</td>
    <td>${esc(p.position)}</td><td>${num(p.toi_min, 1)}</td><td>${num(p.points_per_game)}</td><td>${extra}</td></tr>`;
  const miss = t.missing.map((p) => row(p, p.missed ? `${p.missed} záp.` : "–")).join("");
  const ret = t.returning.map((p) => row(p, `po ${p.missed} záp.`)).join("");
  return `<div class="card"><div class="teams">${esc(name)}</div>
    <div class="sub">${status} · chybí ${pct(t.share, 0)} minut pravidelných hráčů (${t.regulars} pravidelných, okno ${t.window} zápasů)</div>
    ${miss ? `<table class="g"><tr><th>Chybí</th><th>Poz.</th><th>TOI</th><th>Body/z.</th><th>Vynechal</th></tr>${miss}</table>` : '<div class="sub">Všichni pravidelní hrají.</div>'}
    ${ret ? `<table class="g"><tr><th>Vrací se</th><th>Poz.</th><th>TOI</th><th>Body/z.</th><th>Absence</th></tr>${ret}</table>` : ""}</div>`;
}

function lineupBlock(g) {
  if (!g.lineup) return "";
  const move = g.sharp && g.sharp_open
    ? `Ostrý trh od otevření: domácí ${pct(g.sharp_open[0], 0)} → ${pct(g.sharp[0], 0)}, hosté ${pct(g.sharp_open[2], 0)} → ${pct(g.sharp[2], 0)}.`
    : "Pohyb ostrého trhu: chybí.";
  return `<h2>Sestava</h2><div class="cards">${lineupTeam(g.home_name, g.lineup.home)}${lineupTeam(g.away_name, g.lineup.away)}</div>
    <p class="sub">${move} Pravidelný hráč = ≥ 60 % z posledních 20 zápasů a TOI nad mediánem týmu; klíčový = horní třetina TOI. Důvod absence (zranění, trest, škrt) NHL API neuvádí. Absence zatím nevstupují do modelu ani do hintu.</p>`;
}

function card(g) {
  const started = new Date(g.start_at) <= Date.now();
  const score = g.final ? `<span class="score">${g.final.home}:${g.final.away}</span>` : "";
  return `<a class="card" href="#/game/${g.id}">
    <div class="meta"><span>${esc(g.league_name)}</span><span>${started && !g.final ? "hraje se" : fmtTime(g.start_at)}</span></div>
    <div class="teams">${score}${esc(g.home_name)} <span class="vs">–</span> ${esc(g.away_name)}</div>
    ${grid(g)}
    ${hintBox(g)}
    ${lineupTag(g)}
  </a>`;
}

async function renderBoard() {
  const d = await api("/api/board?hours=48");
  const games = d.games.filter((g) => g.status !== "finished" || new Date(g.start_at) > Date.now() - 3 * 3600e3);
  const hints = games.filter((g) => g.hint).length;
  const odds = games.filter((g) => g.sharp).length;
  view.innerHTML = `<h1>Dnes a zítra</h1>
    <p class="sub">${games.length} zápasů v příštích 48 h, ${odds} s ostrým trhem, ${hints} s hintem.
      Pravidlo <code>${esc(d.rule.version)}</code>: CZ kniha je proti ostrému trhu pozadu ≥ ${Math.round(d.rule.min_edge * 100)} p. b. a EV &gt; 0.</p>
    <div class="cards">${games.map(card).join("") || `<p class="muted">Žádné zápasy.</p>`}</div>`;
}

// ---------- Game ----------

function lineChart(history, selIdx) {
  const series = Object.entries(history).filter(([, ps]) => ps.length);
  if (!series.length) return `<p class="muted">Bez historie kurzů.</p>`;
  const W = 640, H = 220, P = { l: 40, r: 10, t: 10, b: 24 };
  let t0 = Infinity, t1 = -Infinity, y0 = 1, y1 = 0;
  for (const [, ps] of series) for (const p of ps) {
    const t = +new Date(p.at); t0 = Math.min(t0, t); t1 = Math.max(t1, t);
    y0 = Math.min(y0, p.fair[selIdx]); y1 = Math.max(y1, p.fair[selIdx]);
  }
  if (t1 === t0) t1 = t0 + 3600e3;
  const pad = Math.max(0.02, (y1 - y0) * 0.15); y0 = Math.max(0, y0 - pad); y1 = Math.min(1, y1 + pad);
  const x = (t) => P.l + ((+new Date(t) - t0) / (t1 - t0)) * (W - P.l - P.r);
  const y = (v) => P.t + (1 - (v - y0) / (y1 - y0)) * (H - P.t - P.b);
  let svg = `<svg viewBox="0 0 ${W} ${H}" role="img" aria-label="Vývoj férové pravděpodobnosti">`;
  for (let k = 0; k <= 4; k++) {
    const v = y0 + ((y1 - y0) * k) / 4;
    svg += `<line x1="${P.l}" x2="${W - P.r}" y1="${y(v)}" y2="${y(v)}" stroke="var(--line)"/>
      <text x="${P.l - 6}" y="${y(v) + 4}" text-anchor="end" font-size="11" fill="var(--muted)">${Math.round(v * 100)} %</text>`;
  }
  svg += `<text x="${P.l}" y="${H - 6}" font-size="11" fill="var(--muted)">${esc(fmtTime(t0))}</text>
    <text x="${W - P.r}" y="${H - 6}" font-size="11" fill="var(--muted)" text-anchor="end">${esc(fmtTime(t1))}</text>`;
  for (const [b, ps] of series) {
    // Step line: a price holds until the next change.
    let d = "";
    ps.forEach((p, i) => {
      d += i === 0 ? `M${x(p.at)},${y(p.fair[selIdx])}` : `H${x(p.at)}V${y(p.fair[selIdx])}`;
    });
    d += `H${x(t1)}`;
    svg += `<path d="${d}" fill="none" stroke="${COLORS[b] || "var(--muted)"}" stroke-width="${b === "sharp" ? 2.5 : 1.75}"/>`;
  }
  svg += `</svg>`;
  const legend = series.map(([b]) => `<span><i style="background:${COLORS[b] || "var(--muted)"}"></i>${esc(book(b))}</span>`).join("");
  return svg + `<div class="legend">${legend}</div>`;
}

function waveSpark(w, color) {
  if (!w || !w.series || !w.series.length) return "";
  const W = 300, H = 60, vs = w.series.map((p) => p.level);
  const lo = Math.min(...vs, 0), hi = Math.max(...vs, 0), r = hi - lo || 1;
  const pts = vs.map((v, i) => `${(i / Math.max(1, vs.length - 1)) * W},${H - ((v - lo) / r) * H}`).join(" ");
  const zero = H - ((0 - lo) / r) * H;
  return `<svg viewBox="0 0 ${W} ${H}" style="width:100%;height:60px"><line x1="0" x2="${W}" y1="${zero}" y2="${zero}" stroke="var(--line)"/>
    <polyline points="${pts}" fill="none" stroke="${color}" stroke-width="2"/></svg>`;
}

async function renderGame(id) {
  const [d, w] = await Promise.all([api(`/api/games/${id}`), api(`/matches/${id}/waves`).catch(() => null)]);
  const g = d.game;
  const books = g.books.map((b) => `<tr><td class="l">${esc(book(b.book))}</td>${SEL.map((s) => `<td>${num(b.prices[s])}</td>`).join("")}
    <td>${b.fair ? SEL.map((s, i) => pct(b.fair[i], 0)).join(" / ") : "–"}</td><td>${b.margin ? pct(b.margin) : "–"}</td><td>${ago(b.at)}</td></tr>`).join("");
  const sharp = g.sharp ? `<tr><td class="l">Ostrý trh (férově)</td><td colspan="3" class="muted">ceny nezobrazujeme</td>
    <td>${SEL.map((s, i) => pct(g.sharp[i], 0)).join(" / ")}</td><td>–</td><td>${ago(g.sharp_at)}</td></tr>` : "";
  let contrib = "";
  if (d.prediction && d.prediction.contributions && d.prediction.contributions.length) {
    contrib = `<h2>Proč model říká, co říká</h2><div class="tbl"><table><tr><th class="l">Vstup</th><th>Strana</th><th>Vliv na domácí (p. b.)</th></tr>
      ${d.prediction.contributions.map((c) => `<tr><td class="l">${esc(c.input)}</td><td>${esc(c.side || "")}</td><td>${pp(c.pp / 100)}</td></tr>`).join("")}</table></div>
      <p class="sub">Model <code>${esc(d.prediction.model_version)}</code>, očekávané góly ${num(d.prediction.lambda_home)} : ${num(d.prediction.lambda_away)}. Na trh v backtestu nestačí (G1, A1).</p>`;
  }
  let waves = "";
  if (w && w.home) {
    const wl = (x) => `úroveň ${num(x.level)}, směr ${x.direction >= 0 ? "▲" : "▼"} ${num(Math.abs(x.direction))}, ${x.games} zápasů`;
    waves = `<h2>Vlna (forma)</h2><div class="cards">
      <div class="card"><div class="teams">${esc(g.home_name)}</div>${waveSpark(w.home, "var(--s-home)")}<div class="sub">${wl(w.home)}</div></div>
      <div class="card"><div class="teams">${esc(g.away_name)}</div>${waveSpark(w.away, "var(--s-away)")}<div class="sub">${wl(w.away)}</div></div></div>
      <p class="sub">${esc(w.note || "")}</p>`;
  }
  view.innerHTML = `<a class="back" href="#/">← Zpět</a>
    <h1>${esc(g.home_name)} – ${esc(g.away_name)}</h1>
    <p class="sub">${esc(g.league_name)} · ${fmtTime(g.start_at)}${g.final ? ` · konec ${g.final.home}:${g.final.away}` : ""}</p>
    ${grid(g)}${hintBox(g)}
    ${lineupBlock(g)}
    <h2>Pohyb trhu: férová pravděpodobnost</h2>
    <div class="seg" id="seg">${SEL.map((s, i) => `<button data-i="${i}" class="${i === 0 ? "on" : ""}">${SEL_LABEL[s]}</button>`).join("")}</div>
    <div class="chart" id="chart">${lineChart(d.history, 0)}</div>
    <p class="sub">Když CZ kniha zaostává za ostrým trhem a ten se pohne, vzniká hrana. Uzavírací ostrý kurz je měřítko CLV.</p>
    <h2>Knihy</h2>
    <div class="tbl"><table><tr><th class="l">Kniha</th><th>1</th><th>X</th><th>2</th><th>Férově 1/X/2</th><th>Marže</th><th>Aktualizace</th></tr>${sharp}${books}</table></div>
    ${contrib}${waves}`;
  document.getElementById("seg").addEventListener("click", (e) => {
    const b = e.target.closest("button"); if (!b) return;
    document.querySelectorAll("#seg button").forEach((x) => x.classList.toggle("on", x === b));
    document.getElementById("chart").innerHTML = lineChart(d.history, +b.dataset.i);
  });
}

// ---------- Eval ----------

async function renderEval() {
  const [d, r] = await Promise.all([api("/api/eval"), api("/api/reports")]);
  const s = d.summary;
  const prog = Math.min(100, (100 * s.n_close) / s.need);
  const rows = d.hints.map((h) => `<tr><td class="l">${fmtTime(h.game.start_at)}</td><td class="l">${esc(h.game.home_name)} – ${esc(h.game.away_name)}</td>
    <td>${SEL_LABEL[h.selection]}</td><td class="l">${esc(book(h.book))}</td><td>${num(h.price)}</td><td>${pct(h.p_sharp)}</td>
    <td>${h.p_close != null ? pct(h.p_close) : "–"}</td><td class="${h.clv > 0 ? "up" : h.clv < 0 ? "down" : ""}">${h.clv != null ? pp(h.clv) + " %" : "čeká"}</td>
    <td>${h.result === "win" ? "výhra" : h.result === "loss" ? "prohra" : "–"}</td></tr>`).join("");
  view.innerHTML = `<h1>Evaluace</h1>
    <p class="sub">Papírový deník pravidla <code>${esc(d.rule.version)}</code>: první cena, kdy zápas pravidlo splní, se zapíše a už se nemění.
      Měří se CLV proti uzavíracímu ostrému trhu. Brána: CLV &gt; 0 s intervalem nad nulou na ≥ ${s.need} hintech.</p>
    <div class="tiles">
      <div class="tile"><div class="k">Hintů</div><div class="v">${s.n}</div><div class="d">${s.n_close} s uzavíracím kurzem</div>
        <div class="bar"><i style="width:${prog}%"></i></div><div class="d">${s.n_close} / ${s.need} do brány</div></div>
      <div class="tile"><div class="k">Průměrné CLV</div><div class="v">${s.n_close ? pp(s.clv) + " %" : "–"}</div>
        <div class="d">${s.n_close > 1 ? `95% interval ${pp(s.clv_lo)} až ${pp(s.clv_hi)} %` : "málo dat"}</div></div>
      <div class="tile"><div class="k">Yield (flat)</div><div class="v">${s.settled ? pp(s.yield) + " %" : "–"}</div>
        <div class="d">${s.settled} vyhodnoceno, ${s.wins} výher</div></div>
      <div class="tile"><div class="k">Brána</div><div class="v">${s.gate_pass ? "prošla" : "ve zkoušce"}</div>
        <div class="d">Yield je šum, rozhoduje CLV.</div></div>
    </div>
    <h2>Deník hintů</h2>
    ${rows ? `<div class="tbl"><table><tr><th class="l">Začátek</th><th class="l">Zápas</th><th>Tip</th><th class="l">Kniha</th><th>Kurz</th><th>Ostrý trh</th><th>Uzavírací</th><th>CLV</th><th>Výsledek</th></tr>${rows}</table></div>`
           : `<p class="muted">Zatím žádný hint. Pravidlo běží s každým stažením kurzů.</p>`}
    <h2>Backtesty a rozhodnutí</h2>
    <p class="sub">Co jsme na historii otestovali: model proti trhu (G1), vrstvy proti trhu (F1, A1). Celé reporty, jak jsou v repu.</p>
    <div class="reports" id="reps">${r.reports.map((x) => `<button data-n="${esc(x.name)}">${esc(x.name.replace(/^RESULTS_/, ""))}</button>`).join("")}</div>
    <div class="md" id="md"><p class="muted">Vyber report.</p></div>`;
  document.getElementById("reps").addEventListener("click", async (e) => {
    const b = e.target.closest("button"); if (!b) return;
    document.querySelectorAll("#reps button").forEach((x) => x.classList.toggle("on", x === b));
    const m = await api(`/api/reports/${encodeURIComponent(b.dataset.n)}`);
    document.getElementById("md").innerHTML = window.marked ? marked.parse(m.markdown) : `<pre>${esc(m.markdown)}</pre>`;
  });
}

// ---------- Data ----------

async function renderData() {
  const d = await api("/api/data");
  if (d.loading) {
    const players = (c.players || []).map((x) => `<tr><td class="l">${esc(x.Slug)}</td><td>${x.Season}/${String((x.Season + 1) % 100).padStart(2, "0")}</td>
    <td>${x.Games}</td><td>${x.Lines}</td><td>${x.Players}</td></tr>`).join("");
  view.innerHTML = `<h1>Data</h1><p class="muted">Počítám přehled (projde celou databázi, trvá to desítky sekund)…</p>`;
    setTimeout(() => { if (location.hash === "#/data") route(); }, 8000);
    return;
  }
  const c = d.coverage;
  const games = c.games.map((g) => `<tr><td class="l">${esc(g.Name)}</td><td>${g.Season}/${String((g.Season + 1) % 100).padStart(2, "0")}</td>
    <td>${g.Games}</td><td>${g.Finished}</td><td>${g.Upcoming}</td><td>${pct(g.Finished ? g.WithGoalies / g.Finished : 0, 0)}</td>
    <td>${g.WithOdds}</td></tr>`).join("");
  const odds = c.odds.map((o) => `<tr><td class="l">${esc(o.Slug)}</td><td class="l">${esc(o.Source)}</td><td class="l">${esc(book(o.Book))}</td>
    <td class="l">${esc(o.SnapshotKind)}</td><td>${o.Games}</td><td>${o.Quotes}</td><td>${ago(o.LastCapture)}</td></tr>`).join("");
  const preds = c.predictions.map((p) => `<tr><td class="l"><code>${esc(p.ModelVersion)}</code></td><td>${p.Predictions}</td><td>${ago(p.LastAt)}</td></tr>`).join("");
  const coaches = c.coaches.map((x) => `<tr><td class="l">${esc(x.Slug)}</td><td>${x.Tenures}</td><td>${x.Coaches}</td></tr>`).join("");
  view.innerHTML = `<h1>Data</h1>
    <p class="sub">Co máme v databázi. Přepočet ${ago(d.at)}. Zdroje zdarma: swehockey.se, NHL API, OddsPapi (free), API-HOCKEY (free).</p>
    <h2>Zápasy</h2>
    <div class="tbl"><table><tr><th class="l">Liga</th><th>Sezona</th><th>Zápasů</th><th>Odehráno</th><th>Nadcházejících</th><th>S brankářem</th><th>S kurzy</th></tr>${games}</table></div>
    <h2>Kurzy 1X2</h2>
    <p class="sub">opening = první cena, t24h a t1h = poslední cena 20 h a 1 h před začátkem, closing = poslední před začátkem, adhoc = předběžná cena nadcházejícího zápasu.</p>
    <div class="tbl"><table><tr><th class="l">Liga</th><th class="l">Zdroj</th><th class="l">Kniha</th><th class="l">Snímek</th><th>Zápasů</th><th>Cen</th><th>Poslední</th></tr>${odds}</table></div>
    <h2>Predikce modelu</h2>
    <div class="tbl"><table><tr><th class="l">Model</th><th>Predikcí</th><th>Poslední</th></tr>${preds}</table></div>
    <h2>Hráči v poli</h2>
    <p class="sub">Řádky z boxscore NHL po zápase: TOI, góly, asistence, +/−, střely, tresty. Základ pro sestavy a absence.</p>
    <div class="tbl"><table><tr><th class="l">Liga</th><th>Sezona</th><th>Zápasů</th><th>Řádků</th><th>Hráčů</th></tr>${players || '<tr><td class="l muted" colspan="5">zatím nic</td></tr>'}</table></div>
    <h2>Trenéři</h2>
    <div class="tbl"><table><tr><th class="l">Liga</th><th>Období</th><th>Trenérů</th></tr>${coaches}</table></div>`;
}

route();

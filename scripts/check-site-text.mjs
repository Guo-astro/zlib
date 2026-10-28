// Checks the website's text layout in a real browser, in English and Chinese,
// across phone, tablet and desktop widths:
//
//   runt        a wrapped block whose last line holds only one or two words
//               (or up to four Han characters) and is under half the widest
//               line: the break the guidelines in docs/site.md rule out
//   split word  a word broken across two lines outside code (Z-|Library), or
//               a Chinese word, as Intl.Segmenter finds it (亚马|逊)
//   forced break  text set to white-space: nowrap that wraps all the same
//   overflow    the page scrolls sideways, or text runs past the viewport
//
// Every install tab, command row and agent tab is visited, since each shows
// its own caption. Drawn terminal content (.win) is skipped: it is a fixed
// grid that scrolls inside its window by design.
//
// It needs a browser, which the module itself does not ship:
//
//   python -m http.server 8000 --directory site      # in another terminal
//   PUPPETEER_CORE=/path/to/node_modules/puppeteer-core \
//   CHROME="/path/to/chrome" node scripts/check-site-text.mjs [url] [widths]
//
// PUPPETEER_CORE is a directory holding an installed puppeteer-core (for
// example from `npm i puppeteer-core` in a scratch folder). The exit status
// is 1 when anything is found.

import { createRequire } from "node:module";
import { join } from "node:path";

const base = (process.argv[2] || "http://127.0.0.1:8000").replace(/\/$/, "");
const widths = (process.argv[3] || "320,360,375,390,414,480,600,768,834,1024,1180,1280,1366,1440,1600,1920")
  .split(",").map(Number);
const pages = ["/", "/send-to-kindle/"];
const langs = ["en", "zh"];

if (!process.env.PUPPETEER_CORE || !process.env.CHROME) {
  console.error("Set PUPPETEER_CORE (a puppeteer-core install) and CHROME (a Chrome or Chromium binary).");
  process.exit(2);
}
const require = createRequire(join(process.env.PUPPETEER_CORE, "package.json"));
const puppeteer = require(process.env.PUPPETEER_CORE);

// Runs in the page: returns the runts and split words visible now.
function scan() {
  const out = [];
  const segmenter = new Intl.Segmenter("zh", { granularity: "word" });
  const shown = (e) => {
    for (let n = e; n; n = n.parentElement) if (getComputedStyle(n).display === "none") return false;
    return e.getClientRects().length > 0;
  };
  const SEL = "main h1, main h2, main h3, main p, main li, main summary, main td, main th, main figcaption, footer a, footer h3";
  for (const e of document.querySelectorAll(SEL)) {
    if (!shown(e) || e.closest(".win, pre, .visually-hidden")) continue;
    if (e.querySelector("p, li, div, table, pre, ul, ol, h3")) continue;
    const pieces = [];
    const walker = document.createTreeWalker(e, NodeFilter.SHOW_TEXT);
    while (walker.nextNode()) {
      const n = walker.currentNode;
      if (!n.textContent.trim() || !shown(n.parentElement)) continue;
      const inCode = n.parentElement.closest("code, kbd");
      for (const m of n.textContent.matchAll(/\S+/g)) {
        const r = document.createRange();
        r.setStart(n, m.index);
        r.setEnd(n, m.index + m[0].length);
        const rects = [...r.getClientRects()].filter((b) => b.width > 0);
        if (!rects.length) continue;
        const han = /[　-鿿]/.test(m[0]);
        const tops = new Set(rects.map((b) => Math.round(b.top)));
        if (tops.size > 1 && han) {
          // a Han run may break between words, never inside one (亚马|逊)
          if (!inCode) {
            let at = m.index;
            for (const { segment } of segmenter.segment(m[0])) {
              const w = document.createRange();
              w.setStart(n, at);
              w.setEnd(n, at + segment.length);
              at += segment.length;
              const ys = new Set([...w.getClientRects()].filter((b) => b.width > 0).map((b) => Math.round(b.top)));
              if (segment.length > 1 && ys.size > 1) out.push({ kind: "split word", text: `${segment} in "${e.innerText.trim().slice(0, 50)}"` });
            }
          }
          // place each character on its own line box
          for (let i = 0; i < m[0].length; i++) {
            const c = document.createRange();
            c.setStart(n, m.index + i);
            c.setEnd(n, m.index + i + 1);
            const b = c.getClientRects()[0];
            if (b) pieces.push({ t: m[0][i], han: true, y: b.top + b.height / 2, l: b.left, r: b.right });
          }
          continue;
        }
        if (tops.size > 1 && !inCode) out.push({ kind: "split word", text: `${m[0]} in "${e.innerText.trim().slice(0, 50)}"` });
        const a = rects[0], z = rects[rects.length - 1];
        pieces.push({ t: m[0], han, y: z.top + z.height / 2, l: a.left, r: z.right });
      }
    }
    if (pieces.length < 2) continue;
    const s = getComputedStyle(e);
    const lh = parseFloat(s.lineHeight) || parseFloat(s.fontSize) * 1.3;
    const lines = [];
    for (const p of pieces.sort((a, b) => a.y - b.y || a.l - b.l)) {
      const line = lines.find((l) => Math.abs(l.y - p.y) < lh * 0.5);
      if (line) line.items.push(p);
      else lines.push({ y: p.y, items: [p] });
    }
    if (lines.length < 2) continue;
    const width = (items) => Math.max(...items.map((p) => p.r)) - Math.min(...items.map((p) => p.l));
    const last = lines[lines.length - 1].items.sort((a, b) => a.l - b.l);
    // a Han piece may hold several characters (site.js splits text at word boundaries)
    const hanChars = last.filter((p) => p.han).reduce((n, p) => n + [...p.t.matchAll(/\p{Script=Han}/gu)].length, 0);
    const words = last.filter((p) => !p.han).length + Math.ceil(hanChars / 2);
    const short = last.every((p) => p.han) ? hanChars <= 4 : words <= 2;
    // a balanced break, where the last line is at least half the widest, is a choice
    if (short && width(last) < Math.max(...lines.map((l) => width(l.items))) * 0.5) {
      out.push({ kind: "runt", text: `"${e.innerText.trim().replace(/\s+/g, " ").slice(0, 60)}" ends on "${last.map((p) => p.t).join(last[0].han ? "" : " ")}"` });
    }
  }
  // text meant for one line (white-space: nowrap) that wraps all the same, as
  // Chrome does at a <wbr>
  for (const e of document.querySelectorAll("main *, footer *")) {
    if (!shown(e) || e.closest(".win, pre") || !getComputedStyle(e).whiteSpace.includes("nowrap")) continue;
    if (![...e.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim())) continue;
    const r = document.createRange();
    r.selectNodeContents(e);
    const mids = [...r.getClientRects()].filter((b) => b.width > 0).map((b) => b.top + b.height / 2);
    if (Math.max(...mids) - Math.min(...mids) > parseFloat(getComputedStyle(e).fontSize) * 0.8) out.push({ kind: "forced break", text: `"${e.textContent.trim().slice(0, 40)}" is nowrap but wraps` });
  }
  // sideways scroll, and text that the page's overflow-x: clip would cut
  const vw = document.documentElement.clientWidth;
  // text kept for screen readers only (clipped to nothing, absolutely placed) is not on screen
  const srOnly = (e) => {
    for (let n = e; n; n = n.parentElement) {
      const cs = getComputedStyle(n);
      if (cs.position === "absolute" && cs.clipPath !== "none") return true;
    }
    return false;
  };
  if (document.documentElement.scrollWidth > vw + 1) out.push({ kind: "overflow", text: `page is ${document.documentElement.scrollWidth}px wide` });
  for (const e of document.querySelectorAll("body *")) {
    if (!shown(e) || e.closest(".win, .visually-hidden") || getComputedStyle(e).position === "absolute" || srOnly(e)) continue;
    if (![...e.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim())) continue;
    if (e.getBoundingClientRect().right > vw + 1) out.push({ kind: "overflow", text: `"${e.textContent.trim().slice(0, 40)}" runs past the viewport` });
  }
  return out;
}

const browser = await puppeteer.launch({ executablePath: process.env.CHROME, headless: true });
const found = new Map();
for (const path of pages) {
  for (const lang of langs) {
    for (const w of widths) {
      const page = await browser.newPage();
      await page.setRequestInterception(true);
      // the star count comes from GitHub; answer it locally
      page.on("request", (r) => r.url().startsWith("https://api.github.com/")
        ? r.respond({ status: 200, contentType: "application/json", body: '{"stargazers_count":1204}', headers: { "access-control-allow-origin": "*" } })
        : r.continue());
      await page.evaluateOnNewDocument((l) => { localStorage.setItem("zlib.lang", l); }, lang);
      await page.setViewport({ width: w, height: 900 });
      await page.goto(base + path, { waitUntil: "networkidle0" });
      // show everything a visitor can reach: revealed sections, open answers
      await page.addStyleTag({ content: ".js-reveal [data-r]{opacity:1!important;animation:none!important}" });
      await page.evaluate(() => { document.querySelectorAll("details").forEach((d) => { d.open = true; }); return document.fonts.ready; });
      const states = [null];
      for (const attr of ["data-inst", "data-cmd", "data-agent"]) {
        for (const v of await page.$$eval(`[${attr}]`, (bs, a) => bs.map((b) => b.getAttribute(a)), attr)) states.push(`[${attr}="${v}"]`);
      }
      for (const state of states) {
        if (state) await page.evaluate((s) => document.querySelector(s).click(), state);
        for (const f of await page.evaluate(scan)) {
          const key = `${f.kind} | ${path} ${lang} | ${f.text}`;
          if (!found.has(key)) found.set(key, new Set());
          found.get(key).add(w);
        }
      }
      await page.close();
    }
  }
}
await browser.close();

for (const [key, at] of found) console.log(`${key}\n    at ${[...at].join(", ")}px`);
console.log(found.size ? `\n${found.size} finding(s)` : "No runts, split words or overflow.");
process.exit(found.size ? 1 : 0);

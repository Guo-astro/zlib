// Behaviour shared by every page: scroll reveals, gliding highlights, word
// boundaries for Chinese line breaks, and cross-fades for theme and language
// changes.
//
// The pages work without this file. Nothing is hidden until this script has
// checked it can show it again (the `js-reveal` class), and with
// prefers-reduced-motion it hides nothing and glides nothing.
(() => {
  const root = document.documentElement;
  const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;

  /* ── reveals ─────────────────────────────────────────────
     A [data-reveal] group arrives once, when it scrolls into view; its
     visible [data-r] items follow one another (CSS reads --i). Items are
     counted at reveal time, so script-rendered ones (the theme swatches)
     and hidden tab panes are handled. */
  const groups = [...document.querySelectorAll("[data-reveal]")];
  if (!reduced && "IntersectionObserver" in window && groups.length) {
    root.classList.add("js-reveal");
    const io = new IntersectionObserver(entries => entries.forEach(e => {
      if (!e.isIntersecting) return;
      const g = e.target;
      [...g.querySelectorAll("[data-r]")]
        .filter(el => el.getClientRects().length)
        .forEach((el, i) => el.style.setProperty("--i", Math.min(i, 8)));
      g.dataset.revealed = "";
      io.unobserve(g);
      // Once every item has arrived the group is done, so later changes
      // inside it (a tab switching panes) use their own, shorter motion.
      setTimeout(() => { g.dataset.revealed = "done"; }, 2000);
    }), { rootMargin: "0px 0px -8% 0px" });
    groups.forEach(g => io.observe(g));
  }

  /* ── glides ──────────────────────────────────────────────
     One highlight per box that slides to the current item: the selected
     tab or command, or the hovered nav link. Without this script the
     selected item keeps its own background (.has-glide is never set). */
  const glide = (box, current, { hover = false } = {}) => {
    const g = document.createElement("span");
    g.className = "glide";
    g.setAttribute("aria-hidden", "true");
    box.prepend(g);
    box.classList.add("has-glide");
    let shown = false;
    let at = null; // where the highlight sits now: x, y, w, h
    // FLIP: take the new size at once, then slide and scale back from the old
    // box with transform alone, so nothing is laid out again on every frame
    // and the border and corners end undistorted.
    const place = (el, instant) => {
      if (!el || !el.getClientRects().length) {
        g.style.opacity = "0";
        shown = false;
        return;
      }
      const to = { x: el.offsetLeft, y: el.offsetTop, w: el.offsetWidth, h: el.offsetHeight };
      const settle = `translate(${to.x}px, ${to.y}px)`;
      g.style.transition = "none";
      g.style.width = `${to.w}px`;
      g.style.height = `${to.h}px`;
      if (instant || !shown || !at) {
        g.style.transform = settle;
      } else {
        g.style.transform = `translate(${at.x}px, ${at.y}px) scale(${at.w / to.w}, ${at.h / to.h})`;
        void g.offsetWidth;
        g.style.transition = "";
        g.style.transform = settle;
      }
      void g.offsetWidth;
      g.style.transition = "";
      g.style.opacity = "1";
      at = to;
      shown = true;
    };
    if (hover) {
      box.querySelectorAll("a").forEach(a => a.addEventListener("pointerenter", () => place(a)));
      box.addEventListener("pointerleave", () => {
        g.style.opacity = "0";
        shown = false;
      });
      return;
    }
    place(current(), true);
    new MutationObserver(() => place(current())).observe(box, { subtree: true, attributeFilter: ["aria-selected"] });
    // Labels change width with the language, and the box reflows with the viewport.
    const ro = new ResizeObserver(() => place(current(), true));
    ro.observe(box);
    [...box.children].forEach(c => c !== g && ro.observe(c));
  };
  const selected = box => () => box.querySelector('[aria-selected="true"]');
  document.querySelectorAll(".tabs, .cmd-list").forEach(box => glide(box, selected(box)));
  document.querySelectorAll(".nav-links").forEach(box => glide(box, null, { hover: true }));

  /* ── Chinese line breaks ─────────────────────────────────
     Chinese may wrap between any two characters, so a line can end inside
     a word (亚马|逊). Word boundaries get a <wbr>, and keep-all (CSS, under
     .zh-words) rules out every other break between characters. A word of
     one character stays with the word before it, or, opening a phrase (会
     after a comma, 给 before a command), with what follows it. Without
     Intl.Segmenter, or without this script, the browser's own breaks apply.
     Drawn terminals (.win), code and text kept on one line are left alone:
     Chrome breaks at a <wbr> even under white-space: nowrap. */
  const han = /\p{Script=Han}/u;
  const segmenter = "Segmenter" in Intl && new Intl.Segmenter("zh", { granularity: "word" });
  const skip = "code, kbd, pre, .win, .nowrap, script, style";
  const splitWords = el => {
    const nodes = [];
    const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT);
    while (walker.nextNode()) {
      const n = walker.currentNode;
      if (!han.test(n.data) || n.parentElement.closest(skip)) continue;
      if (getComputedStyle(n.parentElement).whiteSpace.includes("nowrap")) continue;
      nodes.push(n);
    }
    for (const n of nodes) {
      const pieces = [""];
      let last = "";
      let opener = false; // last was a one-character word opening a phrase
      for (let { segment } of segmenter.segment(n.data)) {
        const afterHan = han.test(last.at(-1) || "");
        if (afterHan && han.test(segment[0]) && segment.length > 1 && !opener) pieces.push("");
        if (opener && /^\s+$/.test(segment)) segment = " ";
        pieces[pieces.length - 1] += segment;
        opener = han.test(segment) && segment.length === 1 && !afterHan;
        last = segment;
      }
      if (pieces.length < 2 && pieces[0] === n.data) continue;
      const frag = document.createDocumentFragment();
      pieces.forEach((p, i) => {
        if (i) frag.append(document.createElement("wbr"));
        frag.append(p);
      });
      n.replaceWith(frag);
    }
  };
  if (segmenter) {
    const zh = el => el.querySelectorAll('[data-i18n="zh"]').forEach(splitWords);
    zh(document.body);
    root.classList.add("zh-words");
    // captions written by script later (the command notes) get the same treatment
    new MutationObserver(records => records.forEach(r => {
      if (r.target.closest(skip)) return;
      r.addedNodes.forEach(n => {
        if (n.nodeType !== 1 || n.localName === "wbr") return;
        if (n.matches('[data-i18n="zh"]')) splitWords(n);
        else zh(n);
      });
    })).observe(document.body, { childList: true, subtree: true });
  }

  /* ── cross-fade ──────────────────────────────────────────
     Theme and language changes repaint the whole page; a view transition
     turns that into one short cross-fade where the browser supports it. */
  window.zlibMotion = {
    swap: fn => (!reduced && document.startViewTransition ? document.startViewTransition(fn) : fn()),
  };
})();

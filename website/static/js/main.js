// Dark mode toggle
(function () {
  var toggle = document.getElementById("dark-toggle");
  if (!toggle) return;

  toggle.addEventListener("click", function () {
    var html = document.documentElement;
    var current = html.getAttribute("data-theme");
    var next = current === "dark" ? "light" : "dark";
    html.setAttribute("data-theme", next);
    localStorage.setItem("theme", next);
  });
})();

// Mobile menu toggle + close on outside click
// On docs pages: toggles #sidebar; on all other pages: toggles #nav-mobile-menu
(function () {
  var hamburger = document.getElementById("nav-hamburger");
  if (!hamburger) return;

  var menu = document.getElementById("sidebar") || document.getElementById("nav-mobile-menu");
  if (!menu) return;

  hamburger.addEventListener("click", function () {
    menu.classList.toggle("open");
  });

  document.addEventListener("click", function (e) {
    if (
      menu.classList.contains("open") &&
      !menu.contains(e.target) &&
      !hamburger.contains(e.target)
    ) {
      menu.classList.remove("open");
    }
  });
})();

// TOC: highlight the active section as user scrolls
(function () {
  var tocLinks = document.querySelectorAll(".toc nav#TableOfContents a");
  if (!tocLinks.length) return;

  // Build heading → link map
  var headings = [];
  tocLinks.forEach(function (link) {
    var href = link.getAttribute("href") || "";
    if (!href.startsWith("#")) return;
    var el = document.getElementById(href.slice(1));
    if (el) headings.push({ el: el, link: link });
  });
  if (!headings.length) return;

  function setActive(link) {
    tocLinks.forEach(function (l) { l.classList.remove("toc-active"); });
    if (link) link.classList.add("toc-active");
  }

  // IntersectionObserver: fire when a heading enters the top third of the viewport
  var observer = new IntersectionObserver(
    function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          var match = headings.find(function (h) { return h.el === entry.target; });
          if (match) setActive(match.link);
        }
      });
    },
    { rootMargin: "-8% 0px -75% 0px", threshold: 0 }
  );

  headings.forEach(function (h) { observer.observe(h.el); });
})();

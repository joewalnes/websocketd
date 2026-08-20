function setupSmoothAnchorScrolling() {
    // From http://css-tricks.com/snippets/jquery/smooth-scrolling/
    document.querySelectorAll('a[href*="#"]:not([href="#"])').forEach((link) => {
        link.addEventListener('click', function(ev) {
            if (location.pathname.replace(/^\//, '') == this.pathname.replace(/^\//, '') && location.hostname == this.hostname) {
                // Anchor targets on this page are `<a name="...">`, not id="...", so
                // fall back to a name-selector match if the hash isn't a real id.
                const target = document.querySelector(this.hash) || document.querySelector(`[name="${this.hash.slice(1)}"]`);
                if (target) {
                    window.scrollTo({ top: target.getBoundingClientRect().top + window.pageYOffset, behavior: 'smooth' });
                    ev.preventDefault();
                }
            }
        });
    });
}

function initTabBox(selector) {
    const tabBox = document.querySelector(selector);
    if (!tabBox) {
        return;
    }
    const tabs = [...tabBox.querySelectorAll('.tabs > *')];
    const allContent = [...tabBox.querySelectorAll('.content > *')];

    function showTab(contentId) {
        allContent.forEach((el) => el.style.display = 'none');
        tabs.forEach((el) => el.classList.remove('tab-active'));

        let content = allContent.find((el) => el.classList.contains(contentId));
        let tab = tabs.find((el) => el.dataset.content === contentId);
        if (!content) {
            content = allContent[0];
            tab = tabs[0];
        }
        content.style.display = '';
        tab.classList.add('tab-active');
    }

    tabs.forEach((tab) => {
        tab.addEventListener('click', () => showTab(tab.dataset.content));
    });

    showTab(null);
}

function beginLanguageTicker() {
    const langs = [...document.querySelectorAll('.language-options > li')].map((el) => el.textContent);
    const ticker = document.querySelector('.language-ticker');
    let current = 0;
    setInterval(() => {
        ticker.textContent = langs[current];
        current++;
        if (current == langs.length) {
            current = 0;
        }
    }, 200);
}

function loadGithubStarCount() {
    const el = document.querySelector('#github-star-button');
    if (!el) {
        return;
    }

    const cacheKey = 'websocketd.github-star-count';
    const cacheTtlMs = 60 * 60 * 1000; // 1 hour, to stay well under GitHub's anonymous API rate limit.
    const cached = JSON.parse(localStorage.getItem(cacheKey) || 'null');
    if (cached && Date.now() - cached.fetchedAt < cacheTtlMs) {
        renderStarCount(el, cached.count);
        return;
    }

    fetch('https://api.github.com/repos/joewalnes/websocketd')
        .then((res) => res.ok ? res.json() : Promise.reject(res.status))
        .then((data) => {
            localStorage.setItem(cacheKey, JSON.stringify({ count: data.stargazers_count, fetchedAt: Date.now() }));
            renderStarCount(el, data.stargazers_count);
        })
        .catch(() => {
            // Leave the static "Star on GitHub" fallback text in place.
        });
}

function renderStarCount(el, count) {
    const formatted = count >= 1000 ? `${(count / 1000).toFixed(1)}k` : `${count}`;
    el.innerHTML = `<i class="fa fa-star"></i> ${formatted} Stars`;
}

// home.js is loaded synchronously in <head> with no defer/async, so
// DOMContentLoaded hasn't fired yet when this line runs.
document.addEventListener('DOMContentLoaded', () => {
    setupSmoothAnchorScrolling();
    initTabBox('.tab-box.pkgmgr');
    initTabBox('.tab-box.language');
    beginLanguageTicker();
    loadGithubStarCount();
});

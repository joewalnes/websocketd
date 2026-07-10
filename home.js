function setupSmoothAnchorScolling() {
    // From http://css-tricks.com/snippets/jquery/smooth-scrolling/
    $('a[href*=#]:not([href=#])').click(function() {
        if (location.pathname.replace(/^\//,'') == this.pathname.replace(/^\//,'') && location.hostname == this.hostname) {
            let target = $(this.hash);
            target = target.length ? target : $(`[name=${this.hash.slice(1)}]`);
            if (target.length) {
                $('html,body').animate({
                    scrollTop: target.offset().top
                }, 400);
                return false;
            }
        }
    });
}

function initTabBox(selector) {
    const tabBox = $(selector);
    tabBox.find('.tabs').children().click(function() {
        showTab($(this).data('content'));
    });
    function showTab(contentId) {
        const allContent = tabBox.find('.content').children();
        const allTabs = tabBox.find('.tabs').children();
        allContent.hide();
        allTabs.removeClass('tab-active');
        let content = allContent.filter(`.${contentId}`);
        let tab = allTabs.filter((i, el) => $(el).data('content') === contentId);
        if (!content.length) {
            content = allContent.first();
            tab = allTabs.first();
        }
        content.show();
        tab.addClass('tab-active');
    }
    showTab(null);
}

function beginLanguageTicker() {
    const langs = $('.language-options > li').map((i, e) => $(e).text());
    let current = 0;
    setInterval(() => {
        $('.language-ticker').text(langs[current]);
        current++;
        if (current == langs.length) {
            current = 0;
        }
    }, 200);
}

$(() => {
    setupSmoothAnchorScolling();
    initTabBox('.tab-box.pkgmgr');
    initTabBox('.tab-box.language');
    beginLanguageTicker();
});

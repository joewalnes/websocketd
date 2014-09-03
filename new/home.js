function initTabBox(selector) {
    var tabBox = $(selector);
    tabBox.find('.tabs').children().click(function() {
        showTab($(this).data('content'));
    });
    function showTab(contentId) {
        var allContent = tabBox.find('.content').children(),
            allTabs = tabBox.find('.tabs').children();
        allContent.hide();
        allTabs.removeClass('tab-active');
        var content = allContent.filter('.' + contentId),
            tab = allTabs.filter(function(i, el) { return $(el).data('content') === contentId; });
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
    var langs = $('.language-options > li').map(function(i, e) { return $(e).text() });
    var current = 0;
    setInterval(function() {
        $('.language-ticker').text(langs[current]);
        current++;
        if (current == langs.length) {
            current = 0;
        }
    }, 200);
}

$(function() {
    initTabBox('.tab-box.pkgmgr');
    initTabBox('.tab-box.language');
    beginLanguageTicker();
});

$('#submit').click(function(e) {
    let is_error = false;
    $('.uk-form-danger').removeClass('uk-form-danger');
    $('.uk-alert').remove();
    $('#username, #name').each(function(e, item) {
        if ($(item).val().trim() === "") {
            $(item).addClass("uk-form-danger");
            console.log($(item).parent());
            $("<div class='uk-alert uk-alert-small uk-alert-danger'>此欄位必須輸入！！</div>").appendTo($(item).parent());
            is_error = true;
        }
    });
    let is_new = (parseInt($('#id').val()) === 0);
    if ($('#passwd').val() !== $('#passwdv').val()) {
        $('#passwd').parent().append("<div class='uk-alert uk-alert-small uk-alert-danger'>密碼輸入不一致</div>");
        $('#passwd, #passwdv').addClass(["uk-form-danger"]);
        is_error = true;
    } else if (is_new && $('#passwd').val().trim() === '') {
        $('#passwd').parent().append("<div class='uk-alert uk-alert-small uk-alert-danger'>必須輸入密碼！！</div>");
        $('#passwd, #passwdv').addClass(["uk-form-danger"]);
        is_error = true;
    }
    if (is_error) return false;
});
console.log("Hi, there!!");
$("#tstart, #tend").datetimepicker({
    format: "Y-m-d H:i",
});
// CKEditor
let editor;
InlineEditor.create(document.querySelector('#desc_editor')).then(function(newEditor) {editor = newEditor});
$('#submit').click(function(){
    $('#desc').val(editor.getData());
});
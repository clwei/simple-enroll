function setButtonTitle() {
  var btn_cfg = [
    {cls: '.edit', title: '編輯'},
    {cls: '.info', title: '檢視'},
    {cls: '.trash', title: '刪除'},
  ];
  for (var cfg of btn_cfg) {
    document.querySelectorAll(cfg.cls).forEach(function(item){
      item.title = cfg.title;
    });
  }
}
setButtonTitle();
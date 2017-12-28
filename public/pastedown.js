const Pastedown = {
  editBoxContents: "",
  editBoxDirty: false,

  init() {
    $(window).on("hashchange", () => {
      if (window.location.hash.length <= 1) { return; }
      this.editBoxDirty = false;
      this.editBoxContents = "";
      this.loadRendered();
    });
    $("#edit").on("click", () => {
      if (!$("#edit").hasClass("selected")) {
        this.loadEdit();
      }
    });
    $("#view").on("click", () => {
      if (!$("#view").hasClass("selected")) {
        this.loadRendered();
      }
    });
    $("#new").on("click", () => this.onNew());
    $("#share").on("click", () => this.onShare());
    $("#contents").on("input", e => this.onFileChange(e));
    $("#controls").on("change", "input, select", e => this.onFileChange(e));
    // Make the focusout event fire for this div (non-input element). Nice hack.
    $("#share-message").attr("tabindex", -1);
    $("#share-message").addClass("no-focus-outline");
    $("#share-message").on("focusout", e =>
      // Hack to compensate for the fact that document.activeElement isn't set
      // inside the focusout handler. I was unable to find a better way to
      // ignore focusout events happening within the element (say, when you
      // focus a child) after about an hour of Googling.
      setTimeout((function() {
        if ($(document.activeElement).closest("#share-message").length === 0) {
          $("#share-message").fadeOut("fast");
        }
      }), 0)
    );
    if (!this.isFirefox()) {
      // Unfortunately my current text-editing situation only works on chrome.
      // It relies on being able to create a TextInput event, and initTextEvent
      // isn't implemented in firefox yet.
      $("#contents").on("keydown", "#edit-box", e => this.onEditBoxKeydown(e));
      $("#contents").on("keyup", "#edit-box", e => this.onEditBoxKeyup(e));
    }
    this.loadRendered();
  },

  // Basing our detection on the one feature that we differentiate on.
  isFirefox() { return (document.createEvent("TextEvent").initTextEvent == null); },

  disableControls() {
    $("#left").addClass("disabled");
    $("#right").addClass("disabled");
    $("#share").addClass("disabled");
  },

  enableControls() {
    $("#controls .disabled").removeClass("disabled");
  },

  // spin.js options
  spinnerOptions: {
    lines: 9, // The number of lines to draw
    length: 0, // The length of each line
    width: 8, // The line thickness
    radius: 13, // The radius of the inner circle
    corners: 1, // Corner roundness (0..1)
    rotate: 13, // The rotation offset
    color: "#666", // #rgb or #rrggbb
    speed: 1.4, // Rounds per second
    trail: 64, // Afterglow percentage
    shadow: false, // Whether to render a shadow
    hwaccel: false, // Whether to use hardware acceleration
    className: "spinner", // The CSS class to assign to the spinner
    zIndex: 2e9, // The z-index (defaults to 2000000000)
    top: "auto", // Top position relative to parent in px
    left: "auto" // Left position relative to parent in px
  },

  currentFormat() {
    if (this.editBoxDirty) {
      const format = $("#formatChoice input:checked").val();
      switch(format) {
        case "markdown": case "text":
          return format;
        default:
          return $("#formatChoice option:selected").val();
      }
    } else {
      const id = window.location.hash.slice(1);
      const dotIndex = id.lastIndexOf(".");
      if (dotIndex < 0) {
        return "text";
      }
      return id.slice((dotIndex + 1));
    }
  },

  currentLanguage() {
    const format = this.currentFormat();
    return $(`#language option[value=${format}]`).text();
  },

  currentId() { return window.location.hash.slice(1); },

  startSpinner() {
    const target = $("#contents")[0];
    this.spinner = new Spinner(this.spinnerOptions).spin(target);
  },

  stopSpinner() {
    if (this.spinner != null) { this.spinner.stop(); }
  },

  onFileChange(e) {
    if (this.editBoxDirty) { return; }
    if ($(e.target).is("select") && $("#formatChoice input:checked").val() !== "code") {
      return;
    }
    this.editBoxDirty = true;
    window.location.hash = "";
  },

  prepareForViewChange() {
    this.disableControls();
    $("#contents").empty();
    this.startSpinner();
  },

  // mode is either "view" or "edit".
  afterViewChange(mode) {
    this.stopSpinner();
    this.enableControls();
    if (mode === "view") {
      $("#edit").removeClass("selected");
      $("#view").addClass("selected");
      $("#formatChoice").hide();
      $("#formatText").show();
    } else {
      $("#view").removeClass("selected");
      $("#edit").addClass("selected");
      $("#formatText").hide();
      $("#formatChoice").show();
      $("#edit-box").focus();
    }
  },

  redirectToMainPage() {
    window.location.hash = $("body").attr("data-main-id");
  },

  // Load the pastie specified in the URL fragment.
  loadRendered() {
    const id = this.currentId();
    if (!this.editBoxDirty && (id.length <= 1)) {
      this.redirectToMainPage();
      return;
    }
    const options = {
      success: (data, textStatus, jqXHR) => this.onRenderedSuccess(data, textStatus, jqXHR),
      error: (jqXHR, textStatus, errorThrown) => this.onError(jqXHR, textStatus, errorThrown)
    };
    if (this.editBoxDirty) {
      this.editBoxContents = $("#edit-box").val();
      options.type = "post";
      options.url = "/preview";
      options.data = JSON.stringify({text: this.editBoxContents, format: this.currentFormat()});
    } else {
      options.type = "get";
      options.url = `/files/${id}`;
      options.data = { rendered: true };
    }

    this.prepareForViewChange();
    $.ajax(options);
  },

  loadEdit() {
    this.prepareForViewChange();
    if (this.editBoxDirty) {
      const $editBox = $("<textarea id='edit-box'></textarea>");
      $editBox.text(this.editBoxContents);
      $("#contents").html($editBox);
      this.afterViewChange("edit");
    } else {
      const id = this.currentId();
      if (id.length <= 1) {
        this.redirectToMainPage();
        return;
      }
      $.ajax({
        method: "get",
        url: `/files/${id}`,
        success: (data, textStatus, jqXHR) => this.onEditSuccess(data, textStatus, jqXHR),
        error: (jqXHR, textStatus, errorThrown) => this.onError(jqXHR, textStatus, errorThrown)
      });
    }
  },

  // Replace the current content with a new page
  onRenderedSuccess(data, textStatus, jqXHR) {
    this.afterViewChange("view");
    const format = this.currentFormat();
    switch(format) {
      case "text":
        var $text = $("<pre></pre>");
        $text.html(data);
        $("#contents").html($text);
        $("#contents").attr("data-format", "plain-text");
        $("#format").html("plain text");
        break;
      case "markdown":
        $("#contents").html(data);
        $("#contents").attr("data-format", "markdown");
        $("#format").html(format);
        break;
      default:
        $("#contents").html(data);
        $("#contents").attr("data-format", "code");
        $("#format").html(`code (${this.currentLanguage()})`);
    }
  },

  // Show a text edit box with the current contents inside.
  onEditSuccess(data, textStatus, jqXHR) {
    const format = this.currentFormat();
    switch(format) {
      case "text":
        $("#formatChoice input[value=text]").attr("checked", "checked");
        $("#formatChoice input[value!=text]").removeAttr("checked");
        $("#language").val("");
        break;
      case "markdown":
        $("#formatChoice input[value=markdown]").attr("checked", "checked");
        $("#formatChoice input[value!=markdown]").removeAttr("checked");
        $("#language").val("");
        break;
      default:
        $("#formatChoice input[value=code]").attr("checked", "checked");
        $("#formatChoice input[value!=code]").removeAttr("checked");
        $("#language").val(format);
    }
    const $editBox = $("<textarea id='edit-box'></textarea>");
    $editBox.text(data);
    $("#contents").html($editBox);
    this.afterViewChange("edit");
  },

  // Show an error with the page loading.
  onError(jqXHR, textStatus, errorThrown) {
    let message;
    this.stopSpinner();
    if (errorThrown === "Not Found") {
      message = "No such paste.";
    } else {
      message = "Error loading paste.";
    }
    $("#contents").html(`<div class='error'>${message}</div>`);
  },

  onNew() {
    this.editBoxContents = "";
    this.editBoxDirty = true;
    window.location.hash = "";
    $("#formatChoice input[value=text]").attr("checked", "checked");
    $("#formatChoice input[value!=text]").removeAttr("checked");
    $("#language").val("");
    this.loadEdit();
  },

  showShareMessage() {
    const message = `\
<p>Share this URL:</p><input type="text" class="no-focus-outline" value="${window.location.href}" autofocus />\
`;
    $("#share-message").html(message);
    $("#share-message").fadeIn("fast");
    $("#share-message input").focus();
  },

  onShare() {
    let text;
    if (!this.editBoxDirty) {
      this.showShareMessage();
      $("#view").click();
      return;
    }

    if (this.editBoxDirty && (this.editBoxContents !== "")) {
      text = this.editBoxContents;
    } else {
      text = $("#edit-box").val();
    }

    this.disableControls();
    $("#contents").addClass("disabled");
    this.startSpinner();
    $.ajax({
      type: "put",
      url: "/file",
      data: JSON.stringify({text, format: this.currentFormat()}),
      success: (data, textStatus, jqXHR) => this.onShareSuccess(data, textStatus, jqXHR),
      error: (jqXHR, textStatus, errorThrown) => this.onShareError(jqXHR, textStatus, errorThrown)
    });
  },

  onShareSuccess(data, textStatus, jqXHR) {
    $("#contents").removeClass("disabled");
    window.location.hash = data; // This triggers view to be loaded which stops the spinner and other cleanup.
    this.showShareMessage();
  },

  onShareError(jqXHR, textStatus, errorThrown) {
    this.stopSpinner();
    $(".disabled").removeClass("disabled");
    alert("There was a server error and this paste could not be saved.");
  },

  // http://stackoverflow.com/questions/7553430/javascript-textarea-undo-redo
  // This doesn't work in Firefox.
  insertTextAtCursor($element, text) {
    const event = document.createEvent("TextEvent");
    event.initTextEvent("textInput", true, true, null, text);
    $element.dispatchEvent(event);
  },

  // To make the edit box a little more friendly for typing in code, we make a
  // couple of changes to the default behavior.

  // On 'tab', insert a literal tab rather than the usual browser behavior of
  // going to the next element.
  onEditBoxKeydown(e) {
    if (e.which !== 9) { return; } // tab
    this.onFileChange(e);
    e.preventDefault();
    this.insertTextAtCursor($("#edit-box")[0], "	");
  },

  // On 'return', copy any leading whitespace to the next line (a poor man's
  // auto-indent).
  onEditBoxKeyup(e) {
    if (e.which !== 13) { return; } // enter
    const $t = $("#edit-box");
    // Get the current line.
    const position = $t[0].selectionStart;
    const firstPart = $t.val().substring(0, position - 1);
    const currentLine = firstPart.substring(firstPart.lastIndexOf("\n") + 1);
    // Figure how much leading whitespace is in the current line.
    const match = currentLine.match(/^\s+/);
    const leadingWhitespace = match ? match[0] : "";
    this.insertTextAtCursor($("#edit-box")[0], leadingWhitespace);
  }
};

$(() => Pastedown.init());

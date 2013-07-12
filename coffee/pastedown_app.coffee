Pastedown =
  editBoxContents: ""
  editBoxDirty: false

  init: ->
    $(window).on "hashchange", =>
      return if window.location.hash.length <= 1
      @editBoxDirty = false
      @editBoxContents = ""
      @loadRendered()
    $("#edit").on "click", =>
      return if $("#edit").hasClass("selected")
      @loadEdit()
    $("#view").on "click", =>
      return if $("#view").hasClass("selected")
      @loadRendered()
    $("#new").on "click", => @onNew()
    $("#share").on "click", => @onShare()
    $("#contents").on "input", (e) => @onFileChange(e)
    $("#controls").on "change", "input, select", (e) => @onFileChange(e)
    # Make the focusout event fire for this div (non-input element). Nice hack.
    $("#share-message").attr("tabindex", -1)
    $("#share-message").addClass("no-focus-outline")
    $("#share-message").on "focusout", (e) ->
      # Hack to compensate for the fact that document.activeElement isn't set inside the focusout handler. I
      # was unable to find a better way to ignore focusout events happening within the element (say, when you
      # focus a child) after about an hour of Googling.
      setTimeout (->
        if $(document.activeElement).closest("#share-message").length == 0
          $("#share-message").fadeOut("fast")
      ), 0
    unless @isFirefox()
      # Unfortunately my current text-editing situation only works on chrome. It relies on being able to
      # create a TextInput event, and initTextEvent isn't implemented in firefox yet.
      $("#contents").on "keydown", "#edit-box", (e) => @onEditBoxKeydown(e)
      $("#contents").on "keyup", "#edit-box", (e) => @onEditBoxKeyup(e)

    @loadRendered()

  # Basing our detection on the one feature that we differentiate on
  isFirefox: -> !document.createEvent("TextEvent").initTextEvent?

  disableControls: ->
    $("#left").addClass("disabled")
    $("#right").addClass("disabled")
    $("#share").addClass("disabled")

  enableControls: ->
    $("#controls .disabled").removeClass("disabled")

  # spin.js options
  spinnerOptions:
    lines: 9, # The number of lines to draw
    length: 0, # The length of each line
    width: 8, # The line thickness
    radius: 13, # The radius of the inner circle
    corners: 1, # Corner roundness (0..1)
    rotate: 13, # The rotation offset
    color: "#666", # #rgb or #rrggbb
    speed: 1.4, # Rounds per second
    trail: 64, # Afterglow percentage
    shadow: false, # Whether to render a shadow
    hwaccel: false, # Whether to use hardware acceleration
    className: "spinner", # The CSS class to assign to the spinner
    zIndex: 2e9, # The z-index (defaults to 2000000000)
    top: "auto", # Top position relative to parent in px
    left: "auto" # Left position relative to parent in px

  currentFormat: ->
    if @editBoxDirty
      format = $("#formatChoice input:checked").val()
      if format == "markdown" then format else $("#formatChoice option:selected").val()
    else
      id = window.location.hash[1..]
      dotIndex = id.lastIndexOf(".")
      if dotIndex < 0
        return "text"
      id[(dotIndex + 1)..]

  currentLanguage: ->
    format = @currentFormat()
    $("#language option[value=#{format}]").text()

  currentId: -> window.location.hash[1..]

  startSpinner: ->
    target = $("#contents")[0]
    @spinner = new Spinner(@spinnerOptions).spin(target)

  stopSpinner: ->
    @spinner?.stop()

  onFileChange: (e) ->
    return if @editBoxDirty
    if $(e.target).is("select")
      return unless $("#formatChoice input:checked").val() == "code"
    @editBoxDirty = true
    window.location.hash = ""

  prepareForViewChange: ->
    @disableControls()
    $("#contents").empty()
    @startSpinner()

  # mode is either "view" or "edit".
  afterViewChange: (mode) ->
    @stopSpinner()
    @enableControls()
    if mode == "view"
      $("#edit").removeClass("selected")
      $("#view").addClass("selected")
      $("#formatChoice").hide()
      $("#formatText").show()
    else
      $("#view").removeClass("selected")
      $("#edit").addClass("selected")
      $("#formatText").hide()
      $("#formatChoice").show()
      $("#edit-box").focus()

  redirectToMainPage: ->
    window.location.hash = $("body").attr("data-main-id")

  # Load the pastie specified in the URL fragment.
  loadRendered: ->
    id = @currentId()
    if !@editBoxDirty and id.length <= 1
      @redirectToMainPage()
      return
    options =
      success: (data, textStatus, jqXHR) => @onRenderedSuccess(data, textStatus, jqXHR)
      error: (jqXHR, textStatus, errorThrown) => @onError(jqXHR, textStatus, errorThrown)
    if @editBoxDirty
      @editBoxContents = $("#edit-box").val()
      options.type = "post"
      options.url = "/preview"
      options.data = JSON.stringify(text: @editBoxContents, format: @currentFormat())
    else
      options.type = "get"
      options.url = "/files/#{id}"
      options.data = { rendered: true }

    @prepareForViewChange()
    $.ajax(options)

  loadEdit: ->
    @prepareForViewChange()
    if @editBoxDirty
      $editBox = $("<textarea id='edit-box'></textarea>")
      $editBox.text(@editBoxContents)
      $("#contents").html($editBox)
      @afterViewChange("edit")
    else
      id = @currentId()
      if id.length <= 1
        @redirectToMainPage()
        return
      $.ajax
        method: "get"
        url: "/files/#{id}"
        success: (data, textStatus, jqXHR) => @onEditSuccess(data, textStatus, jqXHR)
        error: (jqXHR, textStatus, errorThrown) => @onError(jqXHR, textStatus, errorThrown)

  # Replace the current content with a new page
  onRenderedSuccess: (data, textStatus, jqXHR) ->
    @afterViewChange("view")
    format = @currentFormat()
    if format == "markdown"
      $("#contents").html(data)
      $("#contents").attr("data-format", "markdown")
      $("#format").html(format)
    else
      $("#contents").html(data)
      $("#contents").attr("data-format", "code")
      $("#format").html("code (#{@currentLanguage()})")
    $("#delete").show()

  # Show a text edit box with the current contents inside.
  onEditSuccess: (data, textStatus, jqXHR) ->
    format = @currentFormat()
    if format == "markdown"
      $("#formatChoice input[value=markdown]").attr("checked", "checked")
      $("#formatChoice input[value!=markdown]").removeAttr("checked")
      $("#language").val("")
    else
      $("#formatChoice input[value=code]").attr("checked", "checked")
      $("#formatChoice input[value!=code]").removeAttr("checked")
      $("#language").val(format)
    $editBox = $("<textarea id='edit-box'></textarea>")
    $editBox.text(data)
    $("#contents").html($editBox)
    @afterViewChange("edit")

  # Show an error with the page loading.
  onError: (jqXHR, textStatus, errorThrown) ->
    @stopSpinner()
    if errorThrown == "Not Found"
      message = "No such paste."
    else
      message = "Error loading paste."
    $("#contents").html("<div class='error'>#{message}</div>")

  onNew: ->
    @editBoxContents = ""
    @editBoxDirty = true
    window.location.hash = ""
    $("#formatChoice input[value=code]").attr("checked", "checked")
    $("#formatChoice input[value!=code]").removeAttr("checked")
    $("#language").val("text")
    @loadEdit()

  showShareMessage: ->
    message = """
      <p>Share this URL:</p><input type="text" class="no-focus-outline" value="#{window.location.href}" autofocus />
      """
    $("#share-message").html(message)
    $("#share-message").fadeIn("fast")
    $("#share-message input").focus()

  onShare: ->
    if !@editBoxDirty
      @showShareMessage()
      $("#view").click()
      return

    if @editBoxDirty and @editBoxContents != ""
      text = @editBoxContents
    else
      text = $("#edit-box").val()

    @disableControls()
    $("#contents").addClass("disabled")
    @startSpinner()
    $.ajax
      type: "put"
      url: "/file"
      data: JSON.stringify(text: text, format: @currentFormat())
      success: (data, textStatus, jqXHR) => @onShareSuccess(data, textStatus, jqXHR)
      error: (jqXHR, textStatus, errorThrown) => @onShareError(jqXHR, textStatus, errorThrown)

  onShareSuccess: (data, textStatus, jqXHR) ->
    $("#contents").removeClass("disabled")
    window.location.hash = data # This triggers view to be loaded which stops the spinner and other cleanup.
    @showShareMessage()

  onShareError: (jqXHR, textStatus, errorThrown) ->
    @stopSpinner()
    $(".disabled").removeClass("disabled")
    alert "There was a server error and this paste could not be saved."

  # http://stackoverflow.com/questions/7553430/javascript-textarea-undo-redo
  # This doesn't work in Firefox.
  insertTextAtCursor: ($element, text) ->
    event = document.createEvent("TextEvent")
    event.initTextEvent("textInput", true, true, null, text)
    $element.dispatchEvent(event)

  # To make the edit box a little more friendly for typing in code, we make a couple of changes to the default
  # behavior.

  # On 'tab', insert a literal tab rather than the usual browser behavior of going to the next element.
  onEditBoxKeydown: (e) ->
    return unless e.which == 9 # tab
    @onFileChange(e)
    e.preventDefault()
    @insertTextAtCursor($("#edit-box")[0], "	")

  # On 'return', copy any leading whitespace to the next line (a poor man's auto-indent)
  onEditBoxKeyup: (e) ->
    return unless e.which == 13 # enter
    $t = $("#edit-box")
    # Get the current line
    position = $t[0].selectionStart
    firstPart = $t.val().substring(0, position - 1)
    currentLine = firstPart.substring(firstPart.lastIndexOf("\n") + 1)
    # Figure how much leading whitespace is in the current line
    leadingWhitespace = currentLine.match(/^\s+/)?[0] or ""
    @insertTextAtCursor($("#edit-box")[0], leadingWhitespace)

$ ->
  Pastedown.init()

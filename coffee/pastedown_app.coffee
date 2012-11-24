Pastedown =
	init: ->
		window.onhashchange = => @loadPastie()
		@loadPastie()

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

	startSpinner: ->
		target = $("#contents")[0]
		@spinner = new Spinner(@spinnerOptions).spin(target)

	stopSpinner: ->
		@spinner?.stop()

	# Load the pastie specified in the URL fragment.
	loadPastie: ->
		$("#controls").addClass("disabled")
		$("#contents").empty()
		@startSpinner()
		id = window.location.hash[1..]
		if id == ""
			window.location.hash = $("body").attr("data-main-id")
			return @loadPastie()
		$.ajax
			method: "get"
			url: "/files/#{id}"
			contentType: "json"
			success: (data, textStatus, jqXHR) => @onSuccess(data, textStatus, jqXHR)
			error: (jqXHR, textStatus, errorThrown) => @onError(jqXHR, textStatus, errorThrown)

	# Replace the current content with a new page
	onSuccess: (data, textStatus, jqXHR) ->
		@stopSpinner()
		$("#controls").removeClass("disabled")
		switch(data.format)
			when "text"
				$text = $("<pre></pre>")
				$text.html(data.contents)
				$("#contents").html($text)
				$("#contents").attr("data-format", "plain-text")
				$("#format").html("plain text")
			when "markdown"
				$("#contents").html(data.contents)
				$("#contents").attr("data-format", "markdown")
				$("#format").html(data.format)
			else
				$("#contents").html(data.contents)
				$("#contents").attr("data-format", "code")
				$("#format").html("code (#{data.format})")

	# Show an error with the page loading.
	onError: (jqXHR, textStatus, errorThrown) ->
		@stopSpinner()
		$("#contents").html("<div class='error'>Error loading file.</div>")

$ ->
	Pastedown.init()

Pastedown =
	# Load the pastie specified in the URL fragment.
	loadPastie: ->
		id = window.location.hash[1..]
		if id == ""
			window.location.hash = $("body").attr("data-main-id")
			return @loadPastie()
		$.ajax
			method: "get"
			url: "/files/#{id}"
			success: (data, textStatus, jqXHR) => @onSuccess(data, textStatus, jqXHR)
			error: (jqXHR, textStatus, errorThrown) => @onError(jqXHR, textStatus, errorThrown)

	# Replace the current content with a new page
	onSuccess: (data, textStatus, jqXHR) ->
		$("#main").html(data)

	# Show an error with the page loading.
	onError: (jqXHR, textStatus, errorThrown) ->
		$("#main").html("<em>Error loading file.</em>")


$ ->
	Pastedown.loadPastie()

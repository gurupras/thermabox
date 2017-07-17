var socket;
window.onload = function() {
	socket = io();

	socket.on('get-temperature', function(msg) {
		console.log("socket.io [get-temperature]: json stringify: " + JSON.stringify(msg));
		$('#currentTemp').text(msg.temp);
	});

	setInterval(function() {
		socket.emit('get-temperature');
		$.get('/get-temperature', function(data) {
			$('#currentTemp').text(data);
		});
	}, 300);

	$('#sync').on('click', function() {
		var data = {
			temp: Number($('#temperature').val()),
			threshold: Number($('#threshold').val()),
		};
		socket.emit('set-limits', data);
		$.post("/set-limits", {
			temperature: data.temp,
			threshold: data.threshold,
		});
	});

	// Update the current limits
	$.get("/get-limits", function(data) {
		var json = JSON.parse(data);
		$('#temperature').val(json.temperature);
		$('#threshold').val(json.threshold);
	});
}

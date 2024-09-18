var conn = new WebSocket(`ws://localhost:8080/`);

conn.onclose = function() {
    alert('Connection lost');
};

conn.onmessage = function(event) {
    // TODO
    console.log(event.data);
};

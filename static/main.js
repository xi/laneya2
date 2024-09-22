var $pre = document.querySelector('pre');
var radius = 5;

var params = new URLSearchParams(location.search);
var gameId = params.get('game');

var socketProtocol = location.protocol.replace('http', 'ws');
var socket = new WebSocket(`${socketProtocol}//${location.host}/ws/${gameId}`);

var send = function(data) {
    socket.send(JSON.stringify(data));
};

var inRect = function(pos, rect, withWalls) {
    if (withWalls) {
        return pos.x >= rect.x1 - 1 && pos.x <= rect.x2 + 1
            && pos.y >= rect.y1 - 1 && pos.y <= rect.y2 + 1;
    } else {
        return pos.x >= rect.x1 && pos.x <= rect.x2
            && pos.y >= rect.y1 && pos.y <= rect.y2;
    }
};

var game = {
    id: -1,
    rects: [],
    seen: {},
    objects: {},

    getRect(pos, withWalls) {
        for (const rect of this.rects) {
            if (inRect(pos, rect, withWalls)) {
                return rect;
            }
        }
    },

    inView(a, b) {
        var dx = a.x - b.x;
        var dy = a.y - b.y;
        return dx * dx + dy * dy < radius * radius;
    },

    getChar(x, y) {
        if (!this.seen[[x, y]]) {
            return [' ', -1];
        }

        var inView = () => Object.values(this.objects).some(
            obj => obj.type === 'player' && this.inView(obj.pos, {x, y})
        );

        if (x === this.ladder.x && y === this.ladder.y) {
            return ['>', inView() ? -1 : 0];
        }
        if (Object.values(this.objects).some(obj => x === obj.pos.x && y === obj.pos.y)) {
            return ['@', 4];
        }
        if (this.getRect({x, y})) {
            return ['.', inView() ? -1 : 0];
        }
        if (this.getRect({x, y}, true)) {
            return ['#', inView() ? -1 : 0];
        }
        return [' ', -1];
    },

    updateSeen(pos) {
        for (let dy = -radius; dy <= radius; dy++) {
            const y = pos.y + dy;
            for (let dx = -radius; dx <= radius; dx++) {
                const x = pos.x + dx;
                if (!this.seen[[x, y]] && this.inView(pos, {x, y})) {
                    this.seen[[x, y]] = true;
                }
            }
        }
    },
};

var getSize = function() {
    // minimum is 10x10
    // maximum is 100x30
    // consider aspect ratio
    // find font size and rows/columns for best match
    // probably have to experiment
    return [10, 20, 100];
};

var render = function() {
    var [fontSize, rows, cols] = getSize();

    var xOffset = -(cols >> 1);
    var yOffset = -(rows >> 1);
    if (game.objects[game.id]) {
        xOffset += game.objects[game.id].pos.x;
        yOffset += game.objects[game.id].pos.y;
    }

    $pre.style.fontSize = fontSize;
    $pre.innerHTML = '';
    var spanColor = -1;
    var span = '';

    var commitSpan = () => {
        if (spanColor === -1) {
            $pre.append(span);
        } else {
            var $span = document.createElement('span');
            $span.innerText = span;
            $span.className = `color-${spanColor}`;
            $pre.append($span);
        }
        span = '';
        spanColor = -1;
    };

    for (let y = 0; y < rows; y++) {
        for (let x = 0; x < cols; x++) {
            const [c, color] = game.getChar(xOffset + x, yOffset + y);
            if (color === spanColor) {
                span += c;
            } else {
                commitSpan();
                span = c;
                spanColor = color;
            }
        }

        commitSpan();
        $pre.append('\n');
    }
};

socket.onclose = function() {
    alert('Connection lost');
};

socket.onmessage = function(event) {
    var messages = JSON.parse(event.data);
    for (const msg of messages) {
        if (msg.action === 'setId') {
            game.id = msg.id;
        } else if (msg.action === 'setLevel') {
            game.rects = msg.rects;
            game.ladder = msg.ladder;
            game.horizontal = msg.horizontal;
            game.vertical = msg.vertical;
            game.seen = {};
        } else if (msg.action === 'create') {
            game.objects[msg.id] = {
                type: msg.type,
                pos: msg.pos,
            };
            if (msg.type === 'player') {
                game.updateSeen(msg.pos);
            }
        } else if (msg.action === 'setPosition') {
            game.objects[msg.id].pos = msg.pos;
            if (game.objects[msg.id].type === 'player') {
                game.updateSeen(msg.pos);
            }
        } else if (msg.action === 'remove') {
            delete game.objects[msg.id];
        } else {
            console.log(msg);
        }
    }
    render();
};

document.onkeydown = function(event) {
    if (event.key === 'ArrowUp') {
        send({action: 'move', dir: 'up'});
    } else if (event.key === 'ArrowRight') {
        send({action: 'move', dir: 'right'});
    } else if (event.key === 'ArrowDown') {
        send({action: 'move', dir: 'down'});
    } else if (event.key === 'ArrowLeft') {
        send({action: 'move', dir: 'left'});
    } else {
        return;
    }
    event.preventDefault();
};

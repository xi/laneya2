import onDPad from './dpad.js';

var $pre = document.querySelector('pre');

var params = new URLSearchParams(location.search);
var gameId = params.get('game');

var socketProtocol = location.protocol.replace('http', 'ws');
var socket = new WebSocket(`${socketProtocol}//${location.host}/ws/${gameId}`);

var send = function(data) {
    socket.send(JSON.stringify(data));
};

var COLORS = {
    'player': 4,
    'monster': 1,
    'pile': 3,
};

var ITEMS = await fetch('/items/').then(r => r.json());

var inRect = function(pos, rect, withWalls) {
    if (withWalls) {
        return pos.x >= rect.x1 - 1 && pos.x <= rect.x2 + 1
            && pos.y >= rect.y1 - 1 && pos.y <= rect.y2 + 1;
    } else {
        return pos.x >= rect.x1 && pos.x <= rect.x2
            && pos.y >= rect.y1 && pos.y <= rect.y2;
    }
};

var binSearch = function(key) {
    var v2 = 2;
    while (key(v2) <= 0) {
        v2 <<= 1;
    }
    var v1 = v2 >> 1;
    while (v2 - v1 > 1) {
        var v = Math.round((v2 + v1) / 2);
        if (key(v) > 0) {
            v2 = v;
        } else {
            v1 = v;
        }
    }
    return v1;
};

var game = {
    id: -1,
    rects: [],
    seen: {},
    objects: {},
    stats: {
        health: 1,
        healthTotal: 1,
        attack: 0,
        defense: 0,
        speed: 0,
        lineOfSight: 0,
    },
    inventory: {},

    getRect(pos, withWalls) {
        for (const rect of this.rects) {
            if (inRect(pos, rect, withWalls)) {
                return rect;
            }
        }
    },

    inView(a, b) {
        // check radius
        var dx = a.x - b.x;
        var dy = a.y - b.y;
        var r = this.stats.lineOfSight;
        if (dx * dx + dy * dy >= r * r) {
            return false;
        }

        // perf: shortcut if in same rect
        for (const rect of this.rects) {
            if (
                inRect(a, rect, true) && inRect(b, rect, true)
                && (inRect(a, rect) || inRect(b, rect))
            ) {
                return true;
            }
        }

        // ray casting
        if (Math.abs(dx) > Math.abs(dy)) {
            const [c, d] = a.x > b.x ? [b, a] : [a, b];
            return [
                [c.y + 0.4, d.y + 0.4],
                [c.y + 0.4, d.y - 0.4],
                [c.y - 0.4, d.y + 0.4],
                [c.y - 0.4, d.y - 0.4],
            ].some(([y1, y2]) => {
                const f = (y2 - y1) / (d.x - c.x);
                for (let x = c.x + 1; x < d.x; x++) {
                    const y = Math.round((x - c.x) * f + y1);
                    if (!this.getRect({x, y})) {
                        return false;
                    }
                }
                return true;
            });
        } else {
            const [c, d] = a.y > b.y ? [b, a] : [a, b];
            return [
                [c.x + 0.4, d.x + 0.4],
                [c.x + 0.4, d.x - 0.4],
                [c.x - 0.4, d.x + 0.4],
                [c.x - 0.4, d.x - 0.4],
            ].some(([x1, x2]) => {
                const f = (x2 - x1) / (d.y - c.y);
                for (let y = c.y + 1; y < d.y; y++) {
                    const x = Math.round((y - c.y) * f + x1);
                    if (!this.getRect({x, y})) {
                        return false;
                    }
                }
                return true;
            });
        }
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
        var objs = Object.values(this.objects).filter(obj => x === obj.pos.x && y === obj.pos.y);
        for (const obj of objs) {
            if (obj.type === 'player' || inView()) {
                return [obj.rune, COLORS[obj.type]];
            }
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
        var r = this.stats.lineOfSight;
        for (let dy = -r; dy <= r; dy++) {
            const y = pos.y + dy;
            for (let dx = -r; dx <= r; dx++) {
                const x = pos.x + dx;
                if (!this.seen[[x, y]] && this.inView(pos, {x, y})) {
                    this.seen[[x, y]] = true;
                }
            }
        }
    },
};

var screen = {
    rows: null,
    cols: null,
    menuOpen: false,
    menuCursor: 0,
    menuOffset: 0,
    menuSelected: null,

    updateSize() {
        this.rows = binSearch(v => {
            $pre.textContent = '\n'.repeat(v);
            return document.documentElement.scrollHeight - document.documentElement.clientHeight;
        });
        this.cols = binSearch(v => {
            $pre.textContent = ' '.repeat(v);
            return document.body.scrollWidth - document.body.clientWidth;
        });
        this.render();
    },

    toggleMenu() {
        if (this.menuOpen) {
            this.menuOpen = false;
        } else {
            this.menuOpen = true;
            this.menuCursor = 0;
            this.menuOffset = 0;
        }
        this.render();
    },

    commitSpan(text, color) {
        if (color === -1) {
            $pre.append(text);
        } else {
            var $span = document.createElement('span');
            $span.innerText = text;
            $span.className = `color-${color}`;
            $pre.append($span);
        }
    },

    table(items, cols) {
        var c = Math.floor(cols / 3);
        var item = ITEMS[this.menuSelected] || {};
        var rows = items.map(([label, key]) => {
            return [label, '' + game.stats[key], '' + (item[key] || ''), item[key]];
        });
        var l1 = Math.max(...rows.map(row => row[0].length));
        var l2 = Math.max(...rows.map(row => row[1].length));
        var l3 = Math.max(...rows.map(row => row[2].length));

        if ((l1 + 2) + l2 + (l3 ? l3 + 3 : 0) + 2 > c) {
            l1 = c - (2 + l2 + (l3 ? l3 + 3 : 0) + 2);
        }
        rows.forEach((row, i) => {
            this.commitSpan((row[0].substr(0, l1) + ':').padEnd(l1 + 2), -1);
            this.commitSpan(row[1].padStart(l2), -1);
            var l = (l1 + 2) + l2;
            if (row[3]) {
                this.commitSpan(' → ', 7);
                this.commitSpan(row[2].padStart(l3), row[3] < 0 ? 1 : 2);
                l += 3 + l3;
            }
            this.commitSpan(' '.repeat(c - l));
            if ((i + 1) % 3 === 0) {
                $pre.append('\n');
            }
        });
    },

    renderHealth() {
        var health = Math.round(game.stats.health / game.stats.healthTotal * this.cols);
        this.commitSpan('='.repeat(health), 1);
        this.commitSpan('='.repeat(this.cols - health), 0);
        $pre.append('\n');
    },

    renderMenu() {
        var rows = this.rows - 4;
        var items = Object.entries(game.inventory);
        items.sort();

        if (this.menuCursor > items.length - 1) {
            this.menuCursor = items.length - 1;
        }
        if (this.menuCursor < 0) {
            this.menuCursor = 0;
        }

        if (this.menuOffset < this.menuCursor - rows + 1) {
            this.menuOffset = this.menuCursor - rows + 1;
        }
        if (this.menuOffset > this.menuCursor) {
            this.menuOffset = this.menuCursor;
        }

        this.menuSelected = items.length ? items[this.menuCursor][0] : null;

        this.table([
            ['Health', 'health'],
            ['Attack', 'attack'],
            ['Sight', 'lineOfSight'],
            ['Max Health', 'healthTotal'],
            ['Defense', 'defense'],
            ['Speed', 'speed'],
        ], this.cols);
        $pre.append('\n');

        for (let i = 0; i < rows; i++) {
            if (i + this.menuOffset < items.length) {
                var [name, count] = items[i + this.menuOffset];
                var line = ` ${count.toString().padStart(2)} ${name}`
                    .padEnd(this.cols).substr(0, this.cols);
                var color = i + this.menuOffset === this.menuCursor ? 'inverse' : -1;
                this.commitSpan(line, color);
            }
            $pre.append('\n');
        }
    },

    renderMap() {
        var xOffset = -(this.cols >> 1);
        var yOffset = -(this.rows >> 1);
        if (game.objects[game.id]) {
            xOffset += game.objects[game.id].pos.x;
            yOffset += game.objects[game.id].pos.y;
        }

        for (let y = 1; y < this.rows; y++) {
            let span = '';
            let spanColor = -1;

            for (let x = 0; x < this.cols; x++) {
                const [c, color] = game.getChar(xOffset + x, yOffset + y);
                if (color === spanColor) {
                    span += c;
                } else {
                    this.commitSpan(span, spanColor);
                    span = c;
                    spanColor = color;
                }
            }

            this.commitSpan(span, spanColor);
            $pre.append('\n');
        }
    },

    render() {
        $pre.innerHTML = '';

        this.renderHealth();
        if (this.menuOpen) {
            this.renderMenu();
        } else {
            this.renderMap();
        }
    },
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
            game.seen = {};
            for (const [id, obj] of Object.entries(game.objects)) {
                if (obj.type !== 'player') {
                    delete game.objects[id];
                }
            }
        } else if (msg.action === 'create') {
            game.objects[msg.id] = {
                type: msg.type,
                pos: msg.pos,
                rune: msg.rune,
            };
            if (msg.type === 'player') {
                game.updateSeen(msg.pos);
            }
        } else if (msg.action === 'setPosition') {
            game.objects[msg.id].pos = msg.pos;
            if (game.objects[msg.id].type === 'player') {
                game.updateSeen(msg.pos);
            }
        } else if (msg.action === 'setStats') {
            game.stats = msg;
        } else if (msg.action === 'remove') {
            delete game.objects[msg.id];
        } else if (msg.action === 'setInventory') {
            if (msg.amount) {
                game.inventory[msg.item] = msg.amount;
            } else {
                delete game.inventory[msg.item];
            }
        } else {
            console.log(msg);
        }
    }
    screen.render();
};

document.onkeydown = function(event) {
    if (screen.menuOpen) {
        if (event.key === 'ArrowUp' || event.key === 'w') {
            screen.menuCursor -= 1;
        } else if (event.key === 'ArrowDown' || event.key === 's') {
            screen.menuCursor += 1;
        } else if (event.key === 'ArrowRight' || event.key === 'd') {
            if (screen.menuSelected) {
                send({action: 'drop', item: screen.menuSelected});
            }
        } else if (event.key === 'q') {
            screen.toggleMenu();
        } else if (event.key === 'Enter' || event.key === 'e') {
            if (screen.menuSelected) {
                send({action: 'use', item: screen.menuSelected});
            }
        } else {
            return;
        }
        screen.render();
    } else {
        if (event.key === 'ArrowUp' || event.key === 'w') {
            send({action: 'move', dir: 'up'});
        } else if (event.key === 'ArrowRight' || event.key === 'd') {
            send({action: 'move', dir: 'right'});
        } else if (event.key === 'ArrowDown' || event.key === 's') {
            send({action: 'move', dir: 'down'});
        } else if (event.key === 'ArrowLeft' || event.key === 'a') {
            send({action: 'move', dir: 'left'});
        } else if (event.key === 'q') {
            screen.toggleMenu();
        } else if (event.key === 'Enter' || event.key === 'e') {
            send({action: 'pickup'});
        } else {
            return;
        }
    }
    event.preventDefault();
};

onDPad(document.querySelector('#dpad'), dir => {
    var keys = {
        'up': 'ArrowUp',
        'right': 'ArrowRight',
        'down': 'ArrowDown',
        'left': 'ArrowLeft',
    };
    document.onkeydown({
        key: keys[dir],
        preventDefault: () => {},
    });
});

onDPad(document.querySelector('#buttons'), dir => {
    var keys = {
        'up': null,
        'right': 'e',
        'down': null,
        'left': 'q',
    };
    document.onkeydown({
        key: keys[dir],
        preventDefault: () => {},
    });
});

screen.updateSize();
window.addEventListener('resize', () => screen.updateSize(), {passive: true});

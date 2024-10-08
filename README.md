Explore caves with your friends.

```
          #..........#
          #..........#
          #..........#
          #....>.....#
###########.@........#
.....................#
###########.......@..#
          #..........#
```

# How to play

Visit `https://cave.ce9e.org/` to start playing. This will generate a random
game ID. You need to share the generated link with your friends so you are all
in the same game.

Your goal is to move deeper into the cave. When all players stand on the ladder
(`>`), you move on to the next level. But beware! Monsters get stronger and
stronger as you venture further into the cave.

Use the arrow keys or WASD to move. You can attack monsters simply by moving in
their direction. Monsters drop items which you can pick up using the `E` key.
You can use items by opening the menu (`Q`), navigate to the item (up/down)
and then use the `E` key. You can also drop items by pressing right instead.

Monsters may drop equipment. You can equip one armor and one weapon at a time.
The effects of items is displayed on the top of the menu.

If you die or get disconnected, all your items drop in your last position. You
can reload the page to respawn and get all of your items back. But if all of
you die at the same time, the game is lost and you have to restart from the
top.

# Architecture

There is a server (written in go) and a web based client. There could be
multiple different clients (the UI is geared towards terminals), but for now
the focus is on a web client because that is easy to use for anyone.

Communication happens via websockets. Messages are encoded as JSON and always
contain an `action`. Additional fields depend on the specific action.

All logic happens in the `Game.run()` goroutine. `Player.readPump()`,
`Player.writePump()`, and `Monster.run()` are additional goroutines, but they
only send messages to the game. This way, there is no risk of concurrency
issues and the game is always in a consistent state.

The only exception is field of view calculation: That happens on the client for
performance reasons.

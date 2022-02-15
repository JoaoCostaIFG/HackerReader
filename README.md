# HackerReader

A **cli** [HackerNews](https://news.ycombinator.com/) reader, using the
[HackerNews API](https://github.com/HackerNews/API).

I mainly created this as a way to play around with
[Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Lip Gloss](https://github.com/charmbracelet/lipgloss), and consequently
[Go](https://go.dev/). It should be noted that I've barely even seen Go code
before, so expect this to not be that clean.

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Glamour](https://github.com/charmbracelet/glamour)
- [HackerNews API](https://github.com/HackerNews/API)
- [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown)
- [JSON parser](https://github.com/buger/jsonparser)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss)

## Controls

- `ctrl+c / q` - quit;
- `down / j` - move cursor down;
- `up / k` - move cursor up;
- `enter / right / l` - go in story (select);
- `left / h` - go back;
- `space` - hide/unhide post;
- `o` - open story URL in browser (if any);
- `O` - open hovered post in browser.

## TODO

- `[x]` - Load comments;
- `[x]` - Navigate comment tree;
- `[x]` - Add ability to open story URLs in the browser;
- `[x]` - Add ability to hide stories/comments;
- `[x]` - Represent poll results;
- `[x]` - Integrate Lip Gloss;
- `[ ]` - Cool loading indicator;
- `[ ]` - More colors;
- `[ ]` - Hovered post border color.

## Note

- Glamour doesn't currently support commonmark escape chars. There's a
  small hack to deal with the most common causes of it. See
  this [issue](https://github.com/charmbracelet/glamour/issues/106);
- I'm still not sure if I'm doing the JSON stuff curently (specially
  the array stuff).

## License

The code present in this repository is licensed under an
[MIT License](./LICENSE).

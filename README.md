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
- `pgdown` - move 10 elements down;
- `pgup` - move 10 elements up;
- `down / j` - move cursor down;
- `up / k` - move cursor up;
- `enter / right / l` - go in story (select);
- `left / h` - go back;
- `space` - hide/unhide post;
- `o` - open story URL in browser (if any);
- `O` - open hovered post in browser;
- `g / home` - go to first post in list;
- `G / end` - go to last post in list;

### Mouse

- Scolling up and down with the mouse is supported;

## TODO

- `[X]` - Load comments;
- `[X]` - Navigate comment tree;
- `[X]` - Add ability to open story URLs in the browser;
- `[X]` - Add ability to hide stories/comments;
- `[X]` - Represent poll results;
- `[X]` - Integrate Lip Gloss;
- `[X]` - Vertical scroll space thingy;
- `[X]` - Better tag for loading;
- `[X]` - Find way to include style JSON file in app (instead of re-reading it
  everytime);
- `[X]` - Cool loading indicator;
- `[X]` - Better pagination;
- `[X]` - Refactor code;
- `[ ]` - Make it so the current post attempts to stay in the middle of page.
- `[ ]` - More colors;
- `[ ]` - Hovered post border color.

### Maybe TODO

- `[ ]` - Deal with title HTML - API says it is possible but I haven't found a
  single example yet);

## Note

- Glamour doesn't currently support commonmark escape chars. There's a small
  hack to deal with the most common causes of it. See this
  [issue](https://github.com/charmbracelet/glamour/issues/106);
- Glamour text wrapping defines when to start wrapping. If we want a to set a
  max width by wrapping, we should do 1 less char. I pass `withWordWrap(w-1)`;
- I'm still not sure if I'm doing the JSON stuff currently (specially the array
  stuff).

## License

The code present in this repository is licensed under an
[MIT License](./LICENSE).

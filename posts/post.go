package posts

type Post struct {
	Hidden      bool // whether the story has been hidden or not
	Id          int  // -1 when not loaded
	By          string
	Time        int
	Timestr     string
	Storytype   string
	Title       string
	Text        string
	Url         string
	Domain      string
	Score       int
	Descendants int
	Kids        []int
	Parts       []int
	Poll        int
	Parent      int
	Dead        bool
	Deleted     bool
}

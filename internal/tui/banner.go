package tui

// Banner is the ASCII app header drawn above every TUI screen, in
// both the styled runtime theme and the plain Ascii theme.
var Banner = []string{
	`            __                       __                      `,
	`   _____   / /_   ____ _   _____    / /__   __  __     ____  `,
	"  / ___/  / __/  / __ `/  / ___/   / //_/  / / / /    / __ \\ ",
	` (__  )  / /_   / /_/ /  / /__    / ,<    / /_/ /    / /_/ / `,
	`/____/   \__/   \__,_/   \___/   /_/|_|   \__,_/    / .___/  `,
	`                                                   /_/       `,
}

// bannerHeight returns the vertical space the banner occupies: art
// lines plus the separator blank line. It is always drawn.
func (m Model) bannerHeight() int {
	return len(Banner) + 1
}

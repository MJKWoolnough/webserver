package main

import (
	"html/template"
	"net/url"
	"strconv"
	"strings"

	"github.com/MJKWoolnough/bbcode"
)

var colours = map[string]string{
	"aliceblue":            "#F0F8FF",
	"antiquewhite":         "#FAEBD7",
	"aqua":                 "#00FFFF",
	"aquamarine":           "#7FFFD4",
	"azure":                "#F0FFFF",
	"beige":                "#F5F5DC",
	"bisque":               "#FFE4C4",
	"black":                "#000000",
	"blanchedalmond":       "#FFEBCD",
	"blue":                 "#0000FF",
	"blueviolet":           "#8A2BE2",
	"brown":                "#A52A2A",
	"burlywood":            "#DEB887",
	"cadetblue":            "#5F9EA0",
	"chartreuse":           "#7FFF00",
	"chocolate":            "#D2691E",
	"coral":                "#FF7F50",
	"cornflowerblue":       "#6495ED",
	"cornsilk":             "#FFF8DC",
	"crimson":              "#DC143C",
	"cyan":                 "#00FFFF",
	"darkblue":             "#00008B",
	"darkcyan":             "#008B8B",
	"darkgoldenrod":        "#B8860B",
	"darkgray":             "#A9A9A9",
	"darkgrey":             "#A9A9A9",
	"darkgreen":            "#006400",
	"darkkhaki":            "#BDB76B",
	"darkmagenta":          "#8B008B",
	"darkolivegreen":       "#556B2F",
	"darkorange":           "#FF8C00",
	"darkorchid":           "#9932CC",
	"darkred":              "#8B0000",
	"darksalmon":           "#E9967A",
	"darkseagreen":         "#8FBC8F",
	"darkslateblue":        "#483D8B",
	"darkslategray":        "#2F4F4F",
	"darkslategrey":        "#2F4F4F",
	"darkturquoise":        "#00CED1",
	"darkviolet":           "#9400D3",
	"deeppink":             "#FF1493",
	"deepskyblue":          "#00BFFF",
	"dimgray":              "#696969",
	"dimgrey":              "#696969",
	"dodgerblue":           "#1E90FF",
	"firebrick":            "#B22222",
	"floralwhite":          "#FFFAF0",
	"forestgreen":          "#228B22",
	"fuchsia":              "#FF00FF",
	"gainsboro":            "#DCDCDC",
	"ghostwhite":           "#F8F8FF",
	"gold":                 "#FFD700",
	"goldenrod":            "#DAA520",
	"gray":                 "#808080",
	"grey":                 "#808080",
	"green":                "#008000",
	"greenyellow":          "#ADFF2F",
	"honeydew":             "#F0FFF0",
	"hotpink":              "#FF69B4",
	"indianred ":           "#CD5C5C",
	"indigo ":              "#4B0082",
	"ivory":                "#FFFFF0",
	"khaki":                "#F0E68C",
	"lavender":             "#E6E6FA",
	"lavenderblush":        "#FFF0F5",
	"lawngreen":            "#7CFC00",
	"lemonchiffon":         "#FFFACD",
	"lightblue":            "#ADD8E6",
	"lightcoral":           "#F08080",
	"lightcyan":            "#E0FFFF",
	"lightgoldenrodyellow": "#FAFAD2",
	"lightgray":            "#D3D3D3",
	"lightgrey":            "#D3D3D3",
	"lightgreen":           "#90EE90",
	"lightpink":            "#FFB6C1",
	"lightsalmon":          "#FFA07A",
	"lightseagreen":        "#20B2AA",
	"lightskyblue":         "#87CEFA",
	"lightslategray":       "#778899",
	"lightslategrey":       "#778899",
	"lightsteelblue":       "#B0C4DE",
	"lightyellow":          "#FFFFE0",
	"lime":                 "#00FF00",
	"limegreen":            "#32CD32",
	"linen":                "#FAF0E6",
	"magenta":              "#FF00FF",
	"maroon":               "#800000",
	"mediumaquamarine":     "#66CDAA",
	"mediumblue":           "#0000CD",
	"mediumorchid":         "#BA55D3",
	"mediumpurple":         "#9370DB",
	"mediumseagreen":       "#3CB371",
	"mediumslateblue":      "#7B68EE",
	"mediumspringgreen":    "#00FA9A",
	"mediumturquoise":      "#48D1CC",
	"mediumvioletred":      "#C71585",
	"midnightblue":         "#191970",
	"mintcream":            "#F5FFFA",
	"mistyrose":            "#FFE4E1",
	"moccasin":             "#FFE4B5",
	"navajowhite":          "#FFDEAD",
	"navy":                 "#000080",
	"oldlace":              "#FDF5E6",
	"olive":                "#808000",
	"olivedrab":            "#6B8E23",
	"orange":               "#FFA500",
	"orangered":            "#FF4500",
	"orchid":               "#DA70D6",
	"palegoldenrod":        "#EEE8AA",
	"palegreen":            "#98FB98",
	"paleturquoise":        "#AFEEEE",
	"palevioletred":        "#DB7093",
	"papayawhip":           "#FFEFD5",
	"peachpuff":            "#FFDAB9",
	"peru":                 "#CD853F",
	"pink":                 "#FFC0CB",
	"plum":                 "#DDA0DD",
	"powderblue":           "#B0E0E6",
	"purple":               "#800080",
	"rebeccapurple":        "#663399",
	"red":                  "#FF0000",
	"rosybrown":            "#BC8F8F",
	"royalblue":            "#4169E1",
	"saddlebrown":          "#8B4513",
	"salmon":               "#FA8072",
	"sandybrown":           "#F4A460",
	"seagreen":             "#2E8B57",
	"seashell":             "#FFF5EE",
	"sienna":               "#A0522D",
	"silver":               "#C0C0C0",
	"skyblue":              "#87CEEB",
	"slateblue":            "#6A5ACD",
	"slategray":            "#708090",
	"slategrey":            "#708090",
	"snow":                 "#FFFAFA",
	"springgreen":          "#00FF7F",
	"steelblue":            "#4682B4",
	"tan":                  "#D2B48C",
	"teal":                 "#008080",
	"thistle":              "#D8BFD8",
	"tomato":               "#FF6347",
	"turquoise":            "#40E0D0",
	"violet":               "#EE82EE",
	"wheat":                "#F5DEB3",
	"white":                "#FFFFFF",
	"whitesmoke":           "#F5F5F5",
	"yellow":               "#FFFF00",
	"yellowgreen":          "#9ACD32",
}

var fonts = map[string]string{
	"georgia":         "Georgia, serif",
	"times":           "\\\"Times New Roman\\\", Times, serif",
	"times new roman": "\\\"Times New Roman\\\", Times, serif",
	"arial":           "Arial, Helvetica, sans-serif",
	"arial black":     "\\\"Arial Black\\\", Gadget, sans-serif",
	"comic sans ms":   "\\\"Comic Sans MS\\\", cursive, sans-serif",
	"comic sans":      "\\\"Comic Sans MS\\\", cursive, sans-serif",
	"impact":          "Impact, Charcoal, sans-serif",
	"verdana":         "Verdana, Geneva, sans-serif",
	"courier":         "\\\"Courier New\\\", Courier, monospace",
	"lucida console":  "\\\"Lucida Console\\\", Monaco, monospace",
}

type BBCodeExporter map[string]func(*bbcode.Tag, bool)

func (b BBCodeExpoerter) Open(t *bbcode.Tag) string {
	return b.export(t, false)
}

func (b BBCodeExporter) Close(t *bbcode.Tag) string {
	return b.export(t, true)
}

func (b BBCodeExporter) export(t *bbcode.Tag, close bool) string {
	if f, ok := b[strings.ToLower(t.Name)]; ok {
		return f(t, close)
	}
	return defaultOut(t, close)
}

var bbCodeExporter = BBCodeToHTML{
	"b": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</strong>"
		}
		return "<strong>"
	},
	"i": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</em>"
		}
		return "<em>"
	},
	"u": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</u>"
		}
		return "<u>"
	},
	"url": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</a>"
		}
		if !t.Closed {
			return defaultOut(t, close)
		}
		var (
			u   url.URL
			err error
		)
		if t.Attribute != "" {
			u, err = url.Parse(t.Attribute)
		} else if len(t.Inner) == 1 && t.Inner[0].Name == "@TEXT@" {
			u, err = url.Parse(t.Inner[0].Attribute)
		} else {
			return defaultOut(t, close)
		}
		if err != nil {
			return defaultOut(t, close)
		}
		return "<a href=\"" + template.HTMLEscapeString(u.String()) + "\">"
	},
	"center": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</span>"
		}
		return "<span style=\"text-align: center\">"
	},
	"left": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</span>"
		}
		return "<span style=\"text-align: left\">"
	},
	"right": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</span>"
		}
		return "<span style=\"text-align: right\">"
	},
	"size": func(t *bbcode.Tag, close bool) string {
		size, err := strconv.Atoi(t.Attribute)
		if err != nil || size < 1 || size > 50 {
			return defaultOut(t, close)
		}
		if close {
			return "</span>"
		}
		return "<span style=\"font-size: " + t.Attribute + "pt\">"
	},
	"colour": func(t *bbcode.Tag, close bool) string {
		var hex string
		if len(t.Attribute) != 0 && t.Attribute[0] == '#' {
			if len(t.Attribute) == 4 || len(t.Attribute) == 7 {
				if _, err := strconv.ParseUint(t.Attribute[1:], 16, 32); err == nil {
					hex = t.Attribute
				}
			}
		} else {
			hex = colours[strings.ToLower(t.Attribute)]
		}
		if hex == "" {
			return defaultOut(t, close)
		}
		if close {
			return "</span>"
		}
		return "<span style=\"" + hex + "\">"
	},
	"color": func(t *bbcode.Tag, close bool) string { return bbCodeExporter["colour"](t, close) },
	"table": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</table>"
		}
		return "<table>"
	},
	"tr": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</tr>"
		}
		return "<tr>"
	},
	"td": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</td>"
		}
		return "<td>"
	},
	"th": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</h>"
		}
		return "<th>"
	},
	"img": func(t *bbcode.Tag, close bool) string {
		if len(t.Inner) != 1 || t.Inner[0].Name != "@TEXT@" {
			return defaultOut(t, close)
		}
		if close {
			return ""
		}
		u, err := url.Parse(t.Inner[0].Attribute)
		if err != nil {
			return defaultOut(t, close)
		}
		return
	},
	"font": func(t *bbcode.Tag, close bool) string {
		font := fonts[t.Attribute]
		if font == "" {
			return defaultOut(t, close)
		}
		if close {
			return "</span>"
		}
		return "<span style=\"font-family: " + font + "\">"
	},
	"list": func(t *bbcode.Tag, close bool) string {
		var listType string
		switch t.Attribute {
		case "1":
			listType = "1"
		case "A":
			listType = "A"
		case "a":
			listType = "a"
		case "I":
			listType = "I"
		case "i":
			listType = "i"
		}
		if close {
			if listType == "" {
				return "</ul>"
			}
			return "</ol>"
		}
		if listType == "" {
			return "<ul>"
		}
		return "<ol type=\"" + listType + "\">"
	},
	"*": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</li>"
		}
		return "<li>"
	},
	"h1": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</h1>"
		}
		return "<h1>"
	},
	"h2": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</h2>"
		}
		return "<h2>"
	},
	"h3": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</h3>"
		}
		return "<h3>"
	},
	"h4": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</h4>"
		}
		return "<h4>"
	},
	"h5": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</h5>"
		}
		return "<h5>"
	},
	"h6": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</h6>"
		}
		return "<h6>"
	},
	"h7": func(t *bbcode.Tag, close bool) string {
		if close {
			return "</h7>"
		}
		return "<h7>"
	},
}

func defaultOut(t *bbcode.Tag, close bool) string {
	if close {
		return bbcode.BBCodeConverter.Open(t)
	}
	return bbcode.BBCodeConverter.Close(t)
}

func bbCodeToHTML(text string) template.HTML {
	tags := bbcode.Parse(text)
	// walk tree and combine tags and remove invalid code
	return template.HTML(tags.Export(bbCodeToHTML))
}

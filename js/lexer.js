exports.lex = lex

function lex(str) {
	var ret = []
	while (str != "") {
		var match
		if (match = str.match(/^\s+/)) {
			// console.log("whitespace: <" + match[0] + ">")
			str = str.substring(match[0].length, str.length)
		}
		else if (match = str.match(/^#.*\n/)) {
			// console.log("comment: " + match[0].trim())
			str = str.substring(match[0].length, str.length)
		}
		else if (match = str.match(/^[_a-zA-Z][_0-9a-zA-Z]*/)) {
			// console.log("bareword: " + match[0])
			ret.push(["BAREWORD", match[0]])
			str = str.substring(match[0].length, str.length)
		}
		else if (match = str.match(/^\d+/)) {
			// console.log("integer: " + match[0])
			ret.push(["INTEGER", parseInt(match[0], 10)])
			str = str.substring(match[0].length, str.length)
		}
		else if (match = str.match(/^[={}\[\],]/)) {
			// console.log("operator: " + match[0])
			ret.push(["OPERATOR", match[0]])
			str = str.substring(match[0].length, str.length)
		}
		else if (str.charAt(0) == '"') {
			var val = ""
			for (var i=1; i<str.length; i++) {
				if (str.charAt(i) == '"') {
					str = str.substring(i+1, str.length)
					break;
				}
				if (str.charAt(i) == '\\') {
					switch (str.charAt(++i)) {
						case 'n':   val += '\n';  break
						case 'r':   val += '\r';  break
						case 't':   val += '\t';  break
						case '"':   val += '"';   break
						case '\\':  val += '\\';  break
						default:
							ret.push(["ERROR", "illegal escape sequence: \"" + str.charAt(i) + "\""])
							return ret
					}
				}
				else val += str.charAt(i)
			}
			ret.push(["STRING", val])
		}
		else if (match = str.match(/^<<([_a-zA-Z][_0-9a-zA-Z]*)/)) {
			var terminator = match[1]
			var endpos = str.indexOf(terminator, match[0].length)
			if (endpos == -1) {
				ret.push(["ERROR", "missing heredoc terminator: \"" + terminator + "\""])
				return ret
			}
			ret.push(["STRING", str.substring(match[0].length, endpos)])
			str = str.substring(endpos + terminator.length, str.length)
		}
		else {
			ret.push(["ERROR", "illegal character: \"" + str.charAt(0) + "\""])
			return ret
		}
	}
	return ret
}

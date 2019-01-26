exports.lex = lex

function lex(str) {
	var ret = []
	while (str != "") {
		var match
		if (match = str.match(/^\s+/)) {
			// console.log("whitespace: <" + match[0] + ">")
			str = str.substr(match[0].length, str.length)
		}
		else if (match = str.match(/^#.*\n/)) {
			// console.log("comment: " + match[0].trim())
			str = str.substr(match[0].length, str.length)
		}
		else if (match = str.match(/^[_a-zA-Z][_0-9a-zA-Z]*/)) {
			// console.log("bareword: " + match[0])
			ret.push(["BAREWORD", match[0]])
			str = str.substr(match[0].length, str.length)
		}
		else if (match = str.match(/^\d+/)) {
			// console.log("integer: " + match[0])
			ret.push(["INTEGER", parseInt(match[0], 10)])
			str = str.substr(match[0].length, str.length)
		}
		else if (match = str.match(/^[={}\[\],]/)) {
			// console.log("operator: " + match[0])
			ret.push(["OPERATOR", match[0]])
			str = str.substr(match[0].length, str.length)
		}
		else if (match = str.match(/^"([^"]*)"/)) {
			// FIXME: escape characters
			// console.log("string: " + match[0])
			ret.push(["STRING", match[1]])
			str = str.substr(match[0].length, str.length)
		}
		else {
			console.log("lexer failure at: " + str.substr(0, 25) + "...")
			ret.push(null)
			return ret
		}
	}
	return ret
}

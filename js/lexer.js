exports.lex = lex

function countNewlines(str, start, end) {
  var ret = 0
  for (var i=start; i<end; i++)
    if (str.charAt(i) == '\n') ret++
  return ret
}

function lex(str) {
  var ret = []
  var linenum = 1
  while (str != "") {
    var match
    // Skip whitespace
    if (match = str.match(/^[ \t\n\r]/)) {
      // console.log("whitespace: <" + match[0] + ">")
      linenum += countNewlines(str, 0, match[0].length)
      str = str.substring(match[0].length, str.length)
    }

    // Skip comments
    else if (match = str.match(/^(?:#|\/\/).*/)) {
      // console.log("comment: " + match[0].trim())
      linenum += countNewlines(str, 0, match[0].length)
      str = str.substring(match[0].length, str.length)
    }

    // Barewords
    else if (match = str.match(/^[_a-zA-Z][_0-9a-zA-Z]*/)) {
      // console.log("bareword: " + match[0])
      ret.push(["BAREWORD", match[0], linenum])
      str = str.substring(match[0].length, str.length)
    }

    // Integers
    else if (match = str.match(/^\d+/)) {
      // console.log("integer: " + match[0])
      ret.push(["INTEGER", parseInt(match[0], 10), linenum])
      str = str.substring(match[0].length, str.length)
    }

    // Single-character operators
    else if (match = str.match(/^[={}\[\],]/)) {
      // console.log("operator: " + match[0])
      ret.push(["OPERATOR", match[0], linenum])
      str = str.substring(match[0].length, str.length)
    }

    // Strings, with escape characters
    else if (str.charAt(0) == '"') {
      var val = ""
      var warnedAboutIllegalChar = false
      for (var i=1; i<=str.length; i++) {
        if (i >= str.length) {
          ret.push(["ERROR", "unterminated string literal", linenum])
          return ret
        }
        if (str.charAt(i) == '"') {
          str = str.substring(i+1, str.length)
          break
        }
        if (str.charAt(i) == '\\') {
          switch (str.charAt(++i)) {
            case 'b':   val += '\b';  break
            case 'f':   val += '\f';  break
            case 'n':   val += '\n';  break
            case 'r':   val += '\r';  break
            case 't':   val += '\t';  break
            case '"':   val += '"';   break
            case '\\':  val += '\\';  break
            default:
              ret.push(["ERROR", "illegal escape sequence: \"" + str.charAt(i) + "\"", linenum])
          }
        }
        else if (str.charCodeAt(i) < 32) {
          if (!warnedAboutIllegalChar) {
            ret.push(["ERROR", "control character in string: '\\u00" + str.charCodeAt(i).toString(16).padStart(2, '0') + "'", linenum])
            warnedAboutIllegalChar = true
          }
        }
        else {
          val += str.charAt(i)
        }
      }
      ret.push(["STRING", val, linenum])
    }

    // Heredocs
    else if (match = str.match(/^<<([_a-zA-Z][_0-9a-zA-Z]*)/)) {
      var terminator = match[1]
      var endpos = str.indexOf(terminator, match[0].length)
      if (endpos == -1) {
        ret.push(["ERROR", "missing heredoc terminator: \"" + terminator + "\"", linenum])
        return ret
      }
      ret.push(["STRING", str.substring(match[0].length, endpos), linenum])
      linenum += countNewlines(str, 0, endpos+terminator.length)
      str = str.substring(endpos + terminator.length, str.length)
    }

    // Else fail
    else {
      ret.push(["ERROR", "illegal character: \"" + str.charAt(0) + "\"", linenum])
      return ret
    }
  }
  return ret
}

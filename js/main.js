var fs = require('fs')
var lexer = require('./lexer')
var parser = require('./parser')

if (process.argv.length != 3) {
  console.log("Usage:")
  console.log("  " + process.argv[1] + " filename.workflow")
  return
}
var contents = fs.readFileSync(process.argv[2], "utf8")
var tokens = lexer.lex(contents)
var ast = parser.parseWorkflowFile(tokens, [0])
if (!ast || (ast.length > 0 && !ast[ast.length-1])) {
  return
}

console.log(JSON.stringify(ast, null, 2))

'use strict'

const fs = require('fs')
const lexer = require('./lexer')
const parser = require('./parser')

main()

function main() {
  if (process.argv.length !== 3) {
    console.log('Usage:')
    console.log(`  ${process.argv[1]} filename.workflow`)
    return
  }
  const contents = fs.readFileSync(process.argv[2], 'utf8')
  const tokens = lexer.lex(contents)
  const ast = parser.parseWorkflowFile(tokens, [0])
  if (!ast || (ast.length > 0 && !ast[ast.length - 1])) {
    return
  }

  console.log(JSON.stringify(ast, null, 2))
}

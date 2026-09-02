import fs from 'fs'
import sax from 'sax'
import * as svgo from 'svgo'

sax.MAX_BUFFER_LENGTH = Infinity

const config = {
	plugins: [
		'removeXMLProcInst',
		'removeComments',
		{
			name: 'removeAttrs',
			params: {
				attrs: '.*svg.*'
			}
		}
	]
}

const filepath = process.argv[2]

if (!filepath || filepath == '') {
	console.error('Give me filename')
	process.exit()
}

const svg = fs.readFileSync(filepath, 'utf8')
const { data } = svgo.optimize(svg, config)
fs.writeFileSync(filepath, data)

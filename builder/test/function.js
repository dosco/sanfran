
	var markdown = require( "markdown" ).markdown;
	module.exports = function (req, res) {
		var html = markdown.toHTML( '*Hello World*' );
		res.send(html);
	};
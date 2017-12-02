module.exports = {
	hello: function (req, res) {
	  res.send( `Hello ${req.query.name || 'World'}` );
	},

	bye: function (req, res) {
	  res.send( `Goodbye ${req.query.name || 'World'}` );
	}
}
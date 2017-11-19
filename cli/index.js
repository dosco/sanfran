const fs          = require('fs');
const chalk       = require('chalk');
const clear       = require('clear');
const figlet      = require('figlet');
const inquirer    = require('inquirer');
const request     = require('request-json');
const {execSync}  = require('child_process');

const CLI         = require('clui');
const Spinner     = CLI.Spinner;
const status      = new Spinner('Wait...');

const Preferences = require("preferences");
const prefs       = new Preferences('sanfran');

var host = '';
var serverURL = '';
var client = null;

try {
  host = execSync('minikube ip');
  host = host.slice(0, host.length - 1);
} catch(e) {}

//clear();

console.log(
  chalk.bold.green(
    figlet.textSync('SanFran', { horizontalLayout: 'full' })
  )
);

getCommonParams(function(args){
  var isCRUD = ["Create", "Get", "Update", "Delete"]
    .indexOf(args.action) != -1;

  if (args.server === "Other") {
    host = args.host;
  }

  serverURL = `https://${host}`;
  client = request.createClient(serverURL, {
    rejectUnauthorized: false,
  });

  if (isCRUD) {
    handleActions(args.action, function(info){
      status.start();
      switch(args.action) {
        case "Create":
          fnCreate(info.name, info.filename);
          break;
        case "Get":
          fnGet(info.name);
          break;
        case "Update":
          fnCreate(info.name, info.filename);
          break;
        case "Delete":
          fnDelete(info.name);
          break;
      }
      status.stop();
    });
  }

  if (args.action === "List") {
    status.start();
    fnList();
    status.stop();
  }

});

function getCommonParams(callback) {
  var questions = [
    {
      name: 'server',
      type: 'list',
      choices: ["Minikube IP", "Other"],
      message: 'Sanfran API server:'
    },
    {
      type: 'input',
      name: 'host',
      message: 'Hostname / IP of the Sanfran API server:',
      when: function(args) {
        return args.server === "Other";
      },
      validate: function(value) {
        if (value.length) {
          return true;
        } else {
          return 'Hostname or IP of the Sanfran API server';
        }
      }
    },
    {
      name: 'action',
      type: 'list',
      choices: ["Create", "Get", "Update", "Delete", "List"],
      message: 'Pick an action you want to take:'
    },
  ];

  return inquirer.prompt(questions).then(callback)
}

function handleActions(action, callback) {
  var questions = [
    {
      type: 'input',
      name: 'name',
      message: 'Enter function name:',
      validate: function( value ) {
        if (value.length) {
          return true;
        } else {
          return 'Enter function name';
        }
      }
    },
    {
      type: 'input',
      name: 'filename',
      message: 'Enter filename of the function code:',
      when: function(args) {
        return action === "Create" || action === "Update";
      },
      validate: function( value ) {
        if (value.length) {
          return true;
        } else {
          return 'Enter filename of the function code';
        }
      }
    },
  ];

  return inquirer.prompt(questions).then(callback)
}

function fnCreate(name, filename) {
  var d = {
    function: {
      name: name,
      lang: 'js',
      code: base64_encode(filename),
      package: false,
    },
  };

  var h = function(err, res, body) {
    var url = chalk.underline.bold.green(`${serverURL}/fn/${name}`);
    console.log(">", url, "\n");
  }
  return client.post('/api/v1/fn/create', d, h);
}

function fnGet(name) {
  var d = { name: name };
  var h = function(err, res, body) {
    var obj = JSON.stringify(body, null, 2);
    console.log(obj, "\n");
  }
  return client.post('/api/v1/fn/get', d, h);
}

function fnUpdate(name, filename) {
  var d = {
    function: {
      name: name,
      lang: 'js',
      code: base64_encode(filename),
      package: false,
    }
  };
  var h = function(err, res, body) {
    var url = chalk.underline.bold.green(`${serverURL}/fn/${name}`);
    console.log(">", url, "\n");
  }
  return client.post('/api/v1/fn/update', d, h);
}

function fnDelete(name) {
  var d = { name: name };
  var h = function(err, res, body) {
    console.log(">", "Deleted\n");
  }
  return client.post('/api/v1/fn/delete', d, h);
}

function fnList() {
  var h = function(err, res, obj) {
    console.log(">", "Functions:");
    for (var i = 0; i < obj.names.length; i++) {
      var n = i + 1;
      var name = obj.names[i];
      var url = chalk.underline.bold.green(`${serverURL}/fn/${name}`);

      console.log("-",
        chalk.bold(n + ". " + name),
        url
      );
    }
    console.log("\n");
  }
  return client.post('/api/v1/fn/list', {}, h);
}

function base64_encode(file) {
  var bitmap = fs.readFileSync(file);
  return new Buffer(bitmap).toString('base64');
}


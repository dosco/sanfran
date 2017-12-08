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

var serverURL = '';
var client = null;

try {
  let h = execSync('minikube ip')
  var host = h.slice(0, h.length - 1);
} catch(e) {}

console.log(
  chalk.keyword('white').bold(
    figlet.textSync('SanFran', { horizontalLayout: 'full' })
  )
);

getCommonParams(args => {
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
          fnUpdate(info.name, info.filename);
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
      message: 'Sanfran server:'
    },
    {
      type: 'input',
      name: 'host',
      message: 'Host/IP of the Sanfran server:',
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
      message: 'Do What:'
    },
  ];

  return inquirer.prompt(questions).then(callback)
}

function handleActions(action, callback) {
  var questions = [
    {
      type: 'input',
      name: 'name',
      message: 'Function name:',
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
      message: 'Code Filename:',
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
  if (!fs.existsSync(filename)) {
    console.error("File not found: ", filename)
    return
  }
  var d = {
    function: {
      name: name,
      lang: 'js',
      code: base64_encode(filename),
      package: false,
    },
  };

  var h = function(err, res, body) {
    console.log(err);
    console.log(body);

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
  if (!fs.existsSync(filename)) {
    console.error("File not found: ", filename)
    return
  }
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
  let d = { name: name };
  let h = function(err, res, body) {
    console.log(">", "Deleted\n");
  }
  return client.post('/api/v1/fn/delete', d, h);
}

function fnList() {
  let h = function(err, res, obj) {
    console.log(">", "Functions:");
    if (!obj || !obj.names) {
      return;
    }
    for (let i = 0; i < obj.names.length; i++) {
      let n = i + 1;
      let name = obj.names[i];
      let url = chalk.underline.bold.green(`${serverURL}/fn/${name}`);

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
  let bitmap = fs.readFileSync(file);
  return new Buffer(bitmap).toString('base64');
}


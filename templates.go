package main

const (
	sampleSource = `
// Use Parse.Cloud.define to define as many cloud functions as you want.
// For example:
Parse.Cloud.define("hello", function(request, response) {
  response.success("Hello world!");
});
`

	sampleHTML = `
<html>
  <head>
    <title>My ParseApp site</title>
    <style>
    body { font-family: Helvetica, Arial, sans-serif; }
    div { width: 800px; height: 400px; margin: 40px auto; padding: 20px; border: 2px solid #5298fc; }
    h1 { font-size: 30px; margin: 0; }
    p { margin: 40px 0; }
    em { font-family: monospace; }
    a { color: #5298fc; text-decoration: none; }
    </style>
  </head>
  <body>
    <div>
      <h1>Congratulations! You're already hosting with Parse.</h1>
      <p>To get started, edit this file at <em>public/index.html</em> and start adding static content.</p>
      <p>If you want something a bit more dynamic, delete this file and check out <a href="https://parse.com/docs/hosting_guide#webapp">our hosting docs</a>.</p>
    </div>
  </body>
</html>
`

	sampleAppJS = `
// These two lines are required to initialize Express in Cloud Code.
 express = require('express');
 app = express();

// Global app configuration section
app.set('views', 'cloud/views');  // Specify the folder to find templates
app.set('view engine', 'ejs');    // Set the template engine
app.use(express.bodyParser());    // Middleware for reading request body

// This is an example of hooking up a request handler with a specific request
// path and HTTP verb using the Express routing API.
app.get('/hello', function(req, res) {
  res.render('hello', { message: 'Congrats, you just set up your app!' });
});

// // Example reading from the request query string of an HTTP get request.
// app.get('/test', function(req, res) {
//   // GET http://example.parseapp.com/test?message=hello
//   res.send(req.query.message);
// });

// // Example reading from the request body of an HTTP post request.
// app.post('/test', function(req, res) {
//   // POST http://example.parseapp.com/test (with request body "message=hello")
//   res.send(req.body.message);
// });

// Attach the Express app to Cloud Code.
app.listen();
`

	helloEJS = `
<!DOCTYPE html>
<html>
  <head>
    <title>Sample App</title>
  </head>
  <body>
    <h1>Hello World</h1>
    <p><%= message %></p>
  </body>
</html>
`

	helloJade = `
doctype 5
html
  head
    title Sample App
  body
    h1 Hello World
    p= message
`
)

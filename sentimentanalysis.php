<!DOCTYPE html>

<html>

<head>

    <title>Camunda Sentiment Analysis Form</title>

</head>

<body>

<h1>Camunda Sentiment Analysis Form</h1>

<h2>A form which if given a term you'd like to query on the Camunda Forums, will then run the specified query and return the results to you via email.
Please note that queries take time to run, and may take longer depending on the number of queries in the current que.</h2>

<?php


if($_POST["message"]) {


mail("your@email.address", "Your Camunda Forum Query Results",


$_POST["insert your message here"]. "From: an@email.address");


}


?>


<h1> Camunda Forum Sentiment Analysis Query Request</h1>
<h2>A form to fill out with a query you'd like to run sentiment analysis on.</h2>

<h3> Note that queries take time to process, and will be emailed to you once your query is complete.</h3>

<form action="queryform.php" method="post">

<p>Name:</p>
<p><input name="name" value="Your name"></p>

<p>Query Term to Run Analysis On: </p>
<p><textarea rows="10" cols="20" name="comments">Your comments</textarea></p>

<p>On Which Forum:</p>
<p><input type="radio" name="Camunda Cloud" value="Camunda Cloud"> Camunda Cloud</p>
<p><input type="radio" name="Camunda Platform" value="Camunda Platform"> Camunda Platform</p>
<p><input type="radio" name="BPMN.io" value="BPMN.io"> BPMN.io</p>

<p><input type="submit"></p>

</form>

</body>

</html>
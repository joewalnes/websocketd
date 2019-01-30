(function(){
	var counter = 0;
	var echo = function(){
		if (counter === 10){
			return;
		}

		setTimeout(function(){
			counter++;
			echo();
			process.stdout.write(counter.toString() + "\n");
		}, 500);
	}

	echo();
})();

setTimeout(function () {
	window.scrollTo(0, 1);
}, 1000);

document.addEventListener('touchmove', function () {
	event.preventDefault();
});

var emailTextField = document.getElementById('email'),
	firstNameTextField = document.getElementById('first-name'),
	lastNameTextField = document.getElementById('last-name'),
	emailIsValid = function (email) { 
	    var re = /^([a-zA-Z0-9_\.\-])+\@(([a-zA-Z0-9\-])+\.)+([a-zA-Z0-9]{2,4})+$/;
	    return re.test(email);
	},
	signUpButton = document.getElementById('sign-up'),

	enableSignUpButton = function () {
		signUpButton.disabled = false;
		signUpButton.style.color = 'red';
	},

	disableSignUpButton = function () {
		signUpButton.disabled = true;
		signUpButton.style.color = 'darkgray';
	};


emailTextField.addEventListener('keydown', function (event) {
	setTimeout(function() {
		if (emailIsValid(event.srcElement.value)) {
			enableSignUpButton();
		} else {
			disableSignUpButton();
		}
	}, 100);
});

emailTextField.addEventListener('blur', function (event) {
	window.scrollTo(0, 1);
});

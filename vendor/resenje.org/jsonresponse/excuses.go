package jsonresponse // import "resenje.org/jsonresponse"

import (
	"math/rand"
	"time"
)

var (
	excuses = []string{
		"That’s weird...",
		"It worked yesterday.",
		"How is that possible?",
		"It must be a hardware problem.",
		"There has to be something funky in your data.",
		"I haven’t touched that module in weeks.",
		"You must have the wrong version.",
		"It’s just some unlucky coincidence.",
		"It works, but it hasn’t been tested.",
		"Somebody must have changed my code.",
		"Did you check for a virus on your system?",
		"Even though it doesn’t work, How does it feel?",
		"It works on my machine.",
		"It works for me.",
		"The unit test doesn't cover that eventuality.",
		"I'm surprised that was working at all.",
		"It must be because of a leap year.",
		"It's a third party application issue.",
		"I haven't had any experience with that before.",
		"You must have done something wrong.",
		"I have never seen that before in my life.",
		"That was literally a one in a million error.",
		"That's interesting, how did you manage to make it do that?",
		"That code seemed so simple I didn't think it needed testing.",
		"Well, at least it displays a very pretty error.",
		"I'm not familiar with it so I didn't fix it in case I made it worse.",
		"The project manager told me to do it that way.",
		"That's not a bug it's a configuration issue.",
		"I thought I finished that.",
		"The project manager said no one would want that feature.",
		"Management insisted we wouldn't need to waste our time writing unit tests.",
		"I didn't create that part of the program.",
		"Well done, you found my easter egg!.",
		"Don't worry, that value is only wrong half of the time.",
	}
)

func randomExcuse() string {
	rand.Seed(time.Now().UnixNano())
	return excuses[rand.Intn(len(excuses))]
}

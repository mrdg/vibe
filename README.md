# Vibe

This is a program to create musical patterns using commands like:

    loop kick sam1 1 [60]

This loops a 1-beat pattern named 'kick', which sends note number 60 to the
sampler on every beat. Patterns can updated on the fly by reusing the pattern
name.

Play a hihat every second eighth note (0's are rests):

    load-sound sam1 "./demo/hihat.wav" 61
    loop hats sam1 4 [[0 61] [0 61] [0 61] [0 61]]

Use `{}` to play notes at the same time. A minor C chord on the first beat:

    loop chords syn1 4 [{60 63 67} 0 0 0}

To try it out:

    brew install portaudio
    make demo
    # or start fresh with just the samples in the demo directory loaded
    make new

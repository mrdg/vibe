# Ringo

A drum machine / step sequencer.

To try it out:

    make demo


Try some commands:

```
> a 1 2          # toggle the first two steps of sound a
> a '1 2         # toggle the first two beats of sound a
> a '*           # toggle every beat
> a '*/*         # toggle every 1/8 note (assuming quarter note beats)
> a '*//*        # toggle every 1/16 note (assuming quarter note beats)
> a '1,3//*      # toggle every 1/16 note for beats 1 and 3
> a '*/2         # toggle every 2nd 1/8 note
> mute a b       # mute sounds a and b
> rand a         # generate a random pattern for sound a
> clear b c      # clear the patterns for sound b c
> bpm 130        # change the bpm
> beat 9 8       # set the time signature 9/8
> gain a -4      # apply -4dB gain to sound a
> decay a .2     # set amplitude decay for sound a to .2 seconds
> exit
```

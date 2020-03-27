# Ringo

A drum machine / step sequencer.

To try it out:

    make run

```
> setp A *                    # toggle every beat for sound A
> setp A */*                  # toggle every 8th note
> setp A *//*                 # toggle every 16th note
> setp A 1,3 // *             # toggle every 16th note for beat 1 and 3
> setp A 1-3 // *             # toggle every 16th note for the first 3 beats
> setp A 4 // 3,4             # toggle 3rd and 4th 16th notes of beat 4
> setp A *//* | 4//3          # multiple patterns

> mute B
> rand A
> clear B
> bpm 130
> beat 9/8
> gain A -4                   # apply -4dB gain
> decay A 200ms               # set amplitude decay for sound A to 200ms
```

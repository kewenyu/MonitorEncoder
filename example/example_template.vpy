import vapoursynth as vs
from vapoursynth import core


###INPUTFILE###
path = "00000.m2ts"

###SUBTITLE###
sub = "hard_sub.ass"

src8 = core.lsmas.LWLibavSource(path)
output = core.vsf.TextSub(src8, sub)

###DEBUG###
Debug = 1
if Debug == 1:
    core.std.Interleave([src8, output]).set_output()
else:
    output.set_output()

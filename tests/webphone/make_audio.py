"""Original deterministic synthetic microphone: repeating 440/660/880 Hz bursts."""
from pathlib import Path
import math,struct,wave
path=Path(__file__).parent/'fixtures'/'microphone.wav'
rate=48000
with wave.open(str(path),'wb') as f:
    f.setparams((1,2,rate,0,'NONE','not compressed'))
    for i in range(rate*12):
        phase=(i/rate)%3
        tone=[440,660,880][int(phase)]
        envelope=min(1,(phase%1)*50,(1-phase%1)*50)
        f.writeframesraw(struct.pack('<h',int(9000*envelope*math.sin(2*math.pi*tone*i/rate))))
print(path)

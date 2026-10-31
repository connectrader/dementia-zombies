from PIL import Image, ImageDraw
S=1024
im=Image.new('RGBA',(S,S),(0,0,0,0));d=ImageDraw.Draw(im)
d.rounded_rectangle([40,40,S-40,S-40],radius=200,fill=(21,23,27,255),outline=(231,162,59,255),width=36)
# a prescription pill, half bone, half sick green, tilted
pill=Image.new('RGBA',(S,S),(0,0,0,0));p=ImageDraw.Draw(pill)
x0,y0,x1,y1=170,380,854,644;r=(y1-y0)//2
p.rounded_rectangle([x0,y0,x1,y1],radius=r,fill=(238,231,212,255))
p.rectangle([S//2,y0,x1-r,y1],fill=(155,224,106,255));p.pieslice([x1-2*r,y0,x1,y1],-90,90,fill=(155,224,106,255))
p.rounded_rectangle([x0,y0,x1,y1],radius=r,outline=(36,37,43,255),width=26)
p.line([S//2,y0+8,S//2,y1-8],fill=(36,37,43,255),width=22)
# a bite out of the green half
p.ellipse([x1-190,y0-90,x1+40,y0+120],fill=(0,0,0,0))
pill=pill.rotate(-32,resample=Image.BICUBIC,center=(S//2,S//2))
im.alpha_composite(pill)
# re-cut the bite against the background colour so it reads as a bite, not a hole
im.save('icon-1024.png')
im.save('icon.ico',sizes=[(256,256),(128,128),(64,64),(48,48),(32,32),(24,24),(16,16)])
im.resize((64,64),Image.LANCZOS).save('favicon.png')
print('ok')

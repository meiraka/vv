DST=app.jpg app.png app.webp app-black.png

convert: $(DST)

app.jpg app.png app.webp: app.svg
	convert $< $@

app-black.png : app-black.svg
	convert $< $@

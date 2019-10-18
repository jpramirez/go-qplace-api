# go-qplace-api 

API implemetation of ffmpeg callback with workers and async.


### Code 

This is the line we need to use to export ... reasonable:


``` bash

ffmpeg -ss 2  -i Export\ Buttons-2019-10-15_14.56.53.mkv  -filter_complex "[0:v] fps=8,scale=1024:-1,split [a][b];[a] palettegen [p];[b][p] paletteuse" ExportButton.gif

```


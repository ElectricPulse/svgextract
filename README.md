# Description
- The program inside this folder is used to extract the elements
and their positions from an inkscape svg provided by the graphical designer. (Note. you can easily convert .ai to .inkscape.svg)
- It is meant to ease the process of positioning elements inside the visualization by delegating this task to correct person -> the graphical designer

# Usage
Change the inkscape svg and run the program,
it will generate a folder with all the extracted assets and a yaml file describing them,
this you can import inside the visualization.
- to make the program extract an assets position and image put #save-children or #save in the elements name
- #ignore-grouping makes it so that the group itself is ignored in the stucture if under a #save-children parent, instead its children get saved
- #save to saves the element named that way
- #center-position can be added to any name to save the center of the object not the default top left position

go into src directory and run go run *.go

# Dependencies
- inkscape
- golang

# Todo
- If only one asset gets saved in inkscape, replace the root converting `assets.background` -> `assets: Asset`.

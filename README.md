# Mesh2Distance

A simple tool to generate distance fields 3d textures from an .OBJ input 3D mesh

---

All the options that should be available to the user:
- Mirror modes, for each axis:
- none
- positive including center
- positive excluding center
- negative including center
- negative excluding center
- Output type:
  - u8 (output includes bias and scale)
  - u16	(output includes bias and scale)
- Output resolution: eg 256x256x256

Output should be a binary blob, and a json file including:
- Bounding box
- Scale + bias for each axis
- Distance value scale + bias
- Mirror mode
- Output type
- Output resolution

The error is rounded up when the distance is positive, and down when it's negative.
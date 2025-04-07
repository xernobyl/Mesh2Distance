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
- Output resolution (biggest edge)

Output should be a binary blob (or DDS file), and a json file including:
- Bounding box for mesh and grid
- Distance value min and max
- Mirror mode
- Output type (8 or 16 bits)
- Output resolution (width, height, depth)

There's an half a texel border added on the biggest side of the mesh, and the rest is calculated to fit the model, the output texture should always have cubic texels, as I didn't notice any significant improvement from using POT textures.
The error is rounded up when the distance is positive, and down when it's negative, I think that makes sense.

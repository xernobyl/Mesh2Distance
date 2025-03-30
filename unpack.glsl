/*
Example of how to unpack a value from the generated a texture
*/

uniform float min;
uniform float max;

float unpack(float v) {
  return v * (max - min) + min;
}

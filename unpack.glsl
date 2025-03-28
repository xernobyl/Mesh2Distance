/*
Example of how to unpack a value from the generated a texture
*/

uniform float min;
uniform float max;

float unpackLinear(v float) {
  return v * (max - min) + min;
}

float unpackQuad(v float) {
  float zero = -min / (max - min);    // zero point in [0, 1]
  
  if (v == zero) {
    return 0.0;
  }

  if (v < zero) {
    return (v - zero) * (v - zero) / (zero * zero) * min
  }
  
  return (v - zero) * (v - zero) / ((1.0 - zero) * (1.0 - zero)) * max
}

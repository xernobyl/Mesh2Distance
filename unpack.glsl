/*
Example of how to unpack a value from the generated a texture
*/

struct model {
  vec3 bounding_box_min;
  vec3 bounding_box_max;
  float distance_min;
  float distance_max;
};

float sdModel(vec3 p, sampler3D s, model m) {
  vec3 bounding_box_size = m.bounding_box_max - m.bounding_box_min;
  vec3 bounding_box_center = (m.bounding_box_max + m.bounding_box_min) * 0.5;

  // map coordinates to texture
  vec3 c = (p - m.bounding_box_min) / bounding_box_size * (textureSize(s, 0) - 1.0) + 0.5;
  c = c / textureSize(s, 0);
  float d = texture(s, c).b;

  // unpack
  return d * (m.distance_max - m.distance_min) + m.distance_min;
}

#version 450

layout (set = 0, binding = 1) uniform sampler2D surface;

layout (location = 0) in vec2 inTexCoords;

layout (location = 0) out vec4 outColor;

void main() {
    // Sample from the texture, using an immutable sampler
    outColor = texture(surface, inTexCoords);
}
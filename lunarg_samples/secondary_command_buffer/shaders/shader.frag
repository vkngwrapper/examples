#version 450

layout (binding = 1) uniform sampler2D tex;

layout (location = 0) in vec2 texcoord;

layout (location = 0) out vec4 outColor;

void main() {
    outColor = textureLod(tex, texcoord, 0.0);
}
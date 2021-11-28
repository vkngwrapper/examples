#version 450

layout(binding = 0) uniform buf {
    mat4 modelViewProj;
} ubuf;

layout (location = 0) in vec4 pos;
layout (location = 1) in vec2 inTexCoords;

layout (location = 0) out vec2 outTexCoords;

void main() {
    outTexCoords = inTexCoords;
    gl_Position = ubuf.modelViewProj * pos;
}

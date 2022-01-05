#version 450

layout (set = 0, binding = 0) uniform imageBuffer texels;

layout (location = 0) out vec4 outColor;

vec2 vertices[3];
float r;
float g;
float b;

void main() {
    r = imageLoad(texels, 0).r;
    g = imageLoad(texels, 1).r;
    b = imageLoad(texels, 2).r;

    outColor = vec4(r, g, b, 1.0);
    vertices[0] = vec2(-1.0, -1.0);
    vertices[1] = vec2(1.0, -1.0);
    vertices[2] = vec2(0.0, 1.0);

    gl_Position = vec4(vertices[gl_VertexIndex % 3], 0.0, 1.0);
}
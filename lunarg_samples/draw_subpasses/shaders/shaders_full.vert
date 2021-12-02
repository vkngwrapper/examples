#version 450

layout (location = 0) out vec4 outColor;

void main() {
    outColor = vec4(1.0, 0.1, 0.1, 0.5);
    const vec4 verts[4] = vec4[4](
    vec4(-1, -1, 0.5, 1),
    vec4(1, -1, 0.5, 1),
    vec4(-1, 1, 0.5, 1),
    vec4(1, 1, 0.5, 1));

    gl_Position = verts[gl_VertexIndex];
}

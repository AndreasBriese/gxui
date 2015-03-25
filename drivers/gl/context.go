// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gl

import (
	"github.com/google/gxui/math"

	"github.com/go-gl/gl/v3.2-core/gl"
)

type context struct {
	blitter              *blitter
	resolution           resolution
	stats                contextStats
	textureContexts      map[*texture]*textureContext
	vertexStreamContexts map[*vertexStream]*vertexStreamContext
	indexBufferContexts  map[*indexBuffer]*indexBufferContext
	sizeDips, sizePixels math.Size
	clip                 math.Rect
}

func newContext() *context {
	ctx := &context{
		textureContexts:      make(map[*texture]*textureContext),
		vertexStreamContexts: make(map[*vertexStream]*vertexStreamContext),
		indexBufferContexts:  make(map[*indexBuffer]*indexBufferContext),
	}
	ctx.blitter = newBlitter(ctx, &ctx.stats)
	return ctx
}

func (c *context) destroy() {
	for texture, tc := range c.textureContexts {
		delete(c.textureContexts, texture)
		tc.destroy()
		c.stats.textureCount--
	}
	for stream, sc := range c.vertexStreamContexts {
		delete(c.vertexStreamContexts, stream)
		sc.destroy()
		c.stats.vertexStreamCount--
	}
	for buffer, ic := range c.indexBufferContexts {
		delete(c.indexBufferContexts, buffer)
		ic.destroy()
		c.stats.indexBufferCount--
	}
	c.blitter.destroy(c)
	c.blitter = nil
}

func (c *context) beginDraw(sizeDips, sizePixels math.Size) {
	// Reap any dead textures
	for texture, tc := range c.textureContexts {
		if !texture.alive() {
			delete(c.textureContexts, texture)
			tc.destroy()
			c.stats.textureCount--
		}
	}
	for stream, sc := range c.vertexStreamContexts {
		if !stream.alive() {
			delete(c.vertexStreamContexts, stream)
			sc.destroy()
			c.stats.vertexStreamCount--
		}
	}
	for buffer, ic := range c.indexBufferContexts {
		if !buffer.alive() {
			delete(c.indexBufferContexts, buffer)
			ic.destroy()
			c.stats.indexBufferCount--
		}
	}

	dipsToPixels := float32(sizePixels.W) / float32(sizeDips.W)

	c.sizeDips = sizeDips
	c.sizePixels = sizePixels
	c.resolution = resolution(dipsToPixels*65536 + 0.5)

	c.stats.drawCallCount = 0
	c.stats.timer("Frame").start()
}

func (c *context) endDraw() {
	c.stats.timer("Frame").stop()
	c.stats.frameCount++
}

func (c *context) getOrCreateTextureContext(t *texture) *textureContext {
	tc, found := c.textureContexts[t]
	if !found {
		tc = t.newContext()
		c.textureContexts[t] = tc
		c.stats.textureCount++
	}
	return tc
}

func (c *context) getOrCreateVertexStreamContext(vs *vertexStream) *vertexStreamContext {
	vc, found := c.vertexStreamContexts[vs]
	if !found {
		vc = vs.newContext()
		c.vertexStreamContexts[vs] = vc
		c.stats.vertexStreamCount++
	}
	return vc
}

func (c *context) getOrCreateIndexBufferContext(ib *indexBuffer) *indexBufferContext {
	ic, found := c.indexBufferContexts[ib]
	if !found {
		ic = ib.newContext()
		c.indexBufferContexts[ib] = ic
		c.stats.indexBufferCount++
	}
	return ic
}

func (c *context) apply(ds *drawState) {
	r := ds.ClipPixels
	o := c.clip
	if o != r {
		c.clip = r
		vs := c.sizePixels
		rs := r.Size()
		gl.Scissor(int32(r.Min.X), int32(vs.H)-int32(r.Max.Y), int32(rs.W), int32(rs.H))
	}
}

export namespace main {
	
	export class VideoInfo {
	    fullPath: string;
	    duration: string;
	    frameCount: number;
	    codec: string;
	    size: string;
	
	    static createFrom(source: any = {}) {
	        return new VideoInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fullPath = source["fullPath"];
	        this.duration = source["duration"];
	        this.frameCount = source["frameCount"];
	        this.codec = source["codec"];
	        this.size = source["size"];
	    }
	}

}


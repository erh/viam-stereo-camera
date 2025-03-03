# Module viam-stereo-camera 

Provide a description of the purpose of the module and any relevant information.

## Model erh:viam-stereo-camera:stereo-camera

2 images to pcd

### Configuration
The following attribute template can be used to configure this model:

```json
{
    "left": <left camera>,
    "right": <right camera>,
    
    "distance-meters" : .5,
	"focal-length-pixels" : 500,

	"max-disparity" : 64,
    "min-disparity" : 1,
}
```

## flow-movement-sensor
```json
{
    "left": "cam-left-top",
    "right": "cam-right-top",
    "focal-length" : 30
}
```

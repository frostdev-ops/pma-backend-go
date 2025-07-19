# Mobile Upload Page

## Overview

The PMA system now includes a dedicated mobile-friendly upload page that allows users to easily upload screensaver images from their mobile devices. This page is accessible at `/upload` and provides a streamlined interface optimized for touch devices.

## Features

### Core Functionality
- **Touch-Friendly Interface**: Large buttons and touch-optimized design
- **Drag & Drop Support**: Files can be dropped directly onto the upload area
- **Multiple File Selection**: Upload up to 10 images at once
- **Image Preview**: Shows thumbnails of selected images before upload
- **Progress Tracking**: Visual feedback during upload process

### Mobile-Specific Features
- **Camera Integration**: Direct photo capture using device camera
- **Image Compression**: Client-side compression to reduce upload time
- **Quality Selection**: Choose between Small, Balanced, or High quality compression
- **Responsive Design**: Adapts to different screen sizes and orientations
- **Touch Gestures**: Optimized for touch interaction

### Advanced Features
- **Real-time File Validation**: Checks file types and sizes before upload
- **Error Handling**: Comprehensive error messages and retry functionality
- **Accessibility**: Screen reader support and keyboard navigation
- **Modern Web APIs**: Uses latest browser features for optimal performance

## Usage

### Accessing the Upload Page
Navigate to `http://your-pma-server/upload` in any web browser.

### Uploading Images
1. **Select Files**: Tap the upload area or use the file input
2. **Choose Quality**: Select compression level (Small, Balanced, High)
3. **Preview**: Review selected images and remove any unwanted files
4. **Upload**: Tap the upload button to start the process

### Camera Capture
1. **Enable Camera**: Tap the "Take Photo" button
2. **Grant Permission**: Allow camera access when prompted
3. **Capture**: Frame your shot and tap "Capture"
4. **Review**: The photo will be automatically added to your selection

## Technical Details

### File Support
- **Formats**: JPG, PNG, GIF, WebP
- **Size Limit**: 50MB per image
- **Quantity**: Maximum 10 images per upload

### Compression Options
- **Small (60%)**: Faster upload, smaller file size
- **Balanced (80%)**: Recommended setting for most users
- **High (95%)**: Best quality, larger file size

### Browser Compatibility
- **Desktop**: Chrome, Firefox, Safari, Edge
- **Mobile**: iOS Safari, Chrome Mobile, Samsung Internet
- **Features**: Progressive enhancement ensures basic functionality on all browsers

### Security
- **File Type Validation**: Only image files are accepted
- **Size Limits**: Prevents oversized uploads
- **Public Access**: No authentication required for ease of use
- **Input Sanitization**: Protects against malicious uploads

## Integration

The mobile upload page integrates with the existing screensaver system:

- Uses the same `/api/screensaver/images/upload` endpoint
- Follows existing file validation and storage mechanisms
- Maintains compatibility with the main web interface
- Images appear immediately in the screensaver rotation

## Performance

- **Client-Side Compression**: Reduces bandwidth usage
- **Progressive Loading**: Fast initial page load
- **Efficient Uploads**: Optimized for mobile networks
- **Background Processing**: Non-blocking user interface

## Troubleshooting

### Common Issues
- **Camera Not Working**: Check browser permissions and HTTPS requirement
- **Upload Fails**: Verify file size and type requirements
- **Slow Performance**: Try reducing compression quality
- **Interface Issues**: Ensure JavaScript is enabled

### Error Messages
- **File too large**: Reduce image size or use higher compression
- **Invalid file type**: Only use supported image formats
- **Network error**: Check internet connection and try again
- **Camera denied**: Grant camera permission in browser settings

## Future Enhancements

Planned improvements include:
- **Batch Upload Queue**: Better handling of large uploads
- **Upload Resume**: Resume interrupted uploads
- **PWA Features**: Installable app capabilities
- **Enhanced Compression**: More compression options
- **Cloud Integration**: Direct upload from cloud storage

---

For technical support, refer to the main PMA documentation or contact your system administrator. 